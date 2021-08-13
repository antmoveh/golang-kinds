/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package runc

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
)

type CtAct uint8

const (
	CT_ACT_CREATE CtAct = iota + 1
	CT_ACT_RUN
	CT_ACT_RESTORE
)

var errEmptyID = errors.New("container id cannot be empty")

type runner struct {
	init            bool
	enableSubreaper bool
	shouldDestroy   bool
	detach          bool
	listenFDs       []*os.File
	preserveFDs     int
	pidFile         string
	consoleSocket   string
	container       Container
	action          CtAct
	notifySocket    *notifySocket
	criuOpts        *CriuOpts
	logLevel        string
}

type notifySocket struct {
	socket     *net.UnixConn
	host       string
	socketPath string
}

func (r *runner) run(config *Process) (int, error) {
	var err error
	defer func() {
		if err != nil {
			r.destroy()
		}
	}()
	if err = r.checkTerminal(config); err != nil {
		return -1, err
	}
	process, err := newProcess(*config, r.init, r.logLevel)
	if err != nil {
		return -1, err
	}
	if len(r.listenFDs) > 0 {
		process.Env = append(process.Env, "LISTEN_FDS="+strconv.Itoa(len(r.listenFDs)), "LISTEN_PID=1")
		process.ExtraFiles = append(process.ExtraFiles, r.listenFDs...)
	}
	baseFd := 3 + len(process.ExtraFiles)
	for i := baseFd; i < baseFd+r.preserveFDs; i++ {
		_, err = os.Stat("/proc/self/fd/" + strconv.Itoa(i))
		if err != nil {
			return -1, fmt.Errorf("unable to stat preserved-fd %d (of %d): %w", i-baseFd, r.preserveFDs, err)
		}
		process.ExtraFiles = append(process.ExtraFiles, os.NewFile(uintptr(i), "PreserveFD:"+strconv.Itoa(i)))
	}
	rootuid, err := r.container.Config().HostRootUID()
	if err != nil {
		return -1, err
	}
	rootgid, err := r.container.Config().HostRootGID()
	if err != nil {
		return -1, err
	}
	detach := r.detach || (r.action == CT_ACT_CREATE)
	// Setting up IO is a two stage process. We need to modify process to deal
	// with detaching containers, and then we get a tty after the container has
	// started.
	handler := newSignalHandler(r.enableSubreaper, r.notifySocket)
	tty, err := setupIO(process, rootuid, rootgid, config.Terminal, detach, r.consoleSocket)
	if err != nil {
		return -1, err
	}
	defer tty.Close()

	switch r.action {
	case CT_ACT_CREATE:
		err = r.container.Start(process)
	case CT_ACT_RESTORE:
		err = r.container.Restore(process, r.criuOpts)
	case CT_ACT_RUN:
		err = r.container.Run(process)
	default:
		panic("Unknown action")
	}
	if err != nil {
		return -1, err
	}
	if err = tty.waitConsole(); err != nil {
		r.terminate(process)
		return -1, err
	}
	if err = tty.ClosePostStart(); err != nil {
		r.terminate(process)
		return -1, err
	}
	if r.pidFile != "" {
		if err = createPidFile(r.pidFile, process); err != nil {
			r.terminate(process)
			return -1, err
		}
	}
	status, err := handler.forward(process, tty, detach)
	if err != nil {
		r.terminate(process)
	}
	if detach {
		return 0, nil
	}
	if err == nil {
		r.destroy()
	}
	return status, err
}

func (r *runner) destroy() {
	if r.shouldDestroy {
		destroy(r.container)
	}
}

func destroy(container Container) {
	if err := container.Destroy(); err != nil {
		logrus.Error(err)
	}
}

func (r *runner) terminate(p *Process) {
	_ = p.Signal(unix.SIGKILL)
	_, _ = p.Wait()
}

func (r *runner) checkTerminal(config *Process) error {
	detach := r.detach || (r.action == CT_ACT_CREATE)
	// Check command-line for sanity.
	if detach && config.Terminal && r.consoleSocket == "" {
		return errors.New("cannot allocate tty if runc will detach without setting console socket")
	}
	if (!detach || !config.Terminal) && r.consoleSocket != "" {
		return errors.New("cannot use console socket if runc will not detach or allocate tty")
	}
	return nil
}

func newProcess(p Process, init bool, logLevel string) (*libProcess, error) {
	lp := &libProcess{
		Args: p.Args,
		Env:  p.Env,
		// TODO: fix libcontainer's API to better support uid/gid in a typesafe way.
		User:            fmt.Sprintf("%d:%d", p.User.UID, p.User.GID),
		Cwd:             p.Cwd,
		Label:           p.SelinuxLabel,
		NoNewPrivileges: &p.NoNewPrivileges,
		AppArmorProfile: p.ApparmorProfile,
		Init:            init,
		LogLevel:        logLevel,
	}

	if p.ConsoleSize != nil {
		lp.ConsoleWidth = uint16(p.ConsoleSize.Width)
		lp.ConsoleHeight = uint16(p.ConsoleSize.Height)
	}

	if p.Capabilities != nil {
		lp.Capabilities = &Capabilities{}
		lp.Capabilities.Bounding = p.Capabilities.Bounding
		lp.Capabilities.Effective = p.Capabilities.Effective
		lp.Capabilities.Inheritable = p.Capabilities.Inheritable
		lp.Capabilities.Permitted = p.Capabilities.Permitted
		lp.Capabilities.Ambient = p.Capabilities.Ambient
	}
	for _, gid := range p.User.AdditionalGids {
		lp.AdditionalGroups = append(lp.AdditionalGroups, strconv.FormatUint(uint64(gid), 10))
	}
	for _, rlimit := range p.Rlimits {
		rl, err := createLibContainerRlimit(rlimit)
		if err != nil {
			return nil, err
		}
		lp.Rlimits = append(lp.Rlimits, rl)
	}
	return lp, nil
}

// Process specifies the configuration and IO for a process inside
// a container.
type libProcess struct {
	// The command to be run followed by any arguments.
	Args []string

	// Env specifies the environment variables for the process.
	Env []string

	// User will set the uid and gid of the executing process running inside the container
	// local to the container's user and group configuration.
	User string

	// AdditionalGroups specifies the gids that should be added to supplementary groups
	// in addition to those that the user belongs to.
	AdditionalGroups []string

	// Cwd will change the processes current working directory inside the container's rootfs.
	Cwd string

	// Stdin is a pointer to a reader which provides the standard input stream.
	Stdin io.Reader

	// Stdout is a pointer to a writer which receives the standard output stream.
	Stdout io.Writer

	// Stderr is a pointer to a writer which receives the standard error stream.
	Stderr io.Writer

	// ExtraFiles specifies additional open files to be inherited by the container
	ExtraFiles []*os.File

	// Initial sizings for the console
	ConsoleWidth  uint16
	ConsoleHeight uint16

	// Capabilities specify the capabilities to keep when executing the process inside the container
	// All capabilities not specified will be dropped from the processes capability mask
	Capabilities *Capabilities

	// AppArmorProfile specifies the profile to apply to the process and is
	// changed at the time the process is execed
	AppArmorProfile string

	// Label specifies the label to apply to the process.  It is commonly used by selinux
	Label string

	// NoNewPrivileges controls whether processes can gain additional privileges.
	NoNewPrivileges *bool

	// Rlimits specifies the resource limits, such as max open files, to set in the container
	// If Rlimits are not set, the container will inherit rlimits from the parent process
	Rlimits []Rlimit

	// ConsoleSocket provides the masterfd console.
	ConsoleSocket *os.File

	// Init specifies whether the process is the first process in the container.
	Init bool

	// ops processOperations

	LogLevel string
}

func newSignalHandler(enableSubreaper bool, notifySocket *notifySocket) *signalHandler {
	if enableSubreaper {
		// set us as the subreaper before registering the signal handler for the container
		if err := system.SetSubreaper(1); err != nil {
			logrus.Warn(err)
		}
	}
	// ensure that we have a large buffer size so that we do not miss any signals
	// in case we are not processing them fast enough.
	s := make(chan os.Signal, signalBufferSize)
	// handle all signals for the process.
	signal.Notify(s)
	return &signalHandler{
		signals:      s,
		notifySocket: notifySocket,
	}
}

func createLibContainerRlimit(rlimit POSIXRlimit) (Rlimit, error) {
	rl, err := strToRlimit(rlimit.Type)
	if err != nil {
		return Rlimit{}, err
	}
	return Rlimit{
		Type: rl,
		Hard: rlimit.Hard,
		Soft: rlimit.Soft,
	}, nil
}

type Rlimit struct {
	Type int    `json:"type"`
	Hard uint64 `json:"hard"`
	Soft uint64 `json:"soft"`
}

var rlimitMap = map[string]int{
	"RLIMIT_CPU":        unix.RLIMIT_CPU,
	"RLIMIT_FSIZE":      unix.RLIMIT_FSIZE,
	"RLIMIT_DATA":       unix.RLIMIT_DATA,
	"RLIMIT_STACK":      unix.RLIMIT_STACK,
	"RLIMIT_CORE":       unix.RLIMIT_CORE,
	"RLIMIT_RSS":        unix.RLIMIT_RSS,
	"RLIMIT_NPROC":      unix.RLIMIT_NPROC,
	"RLIMIT_NOFILE":     unix.RLIMIT_NOFILE,
	"RLIMIT_MEMLOCK":    unix.RLIMIT_MEMLOCK,
	"RLIMIT_AS":         unix.RLIMIT_AS,
	"RLIMIT_LOCKS":      unix.RLIMIT_LOCKS,
	"RLIMIT_SIGPENDING": unix.RLIMIT_SIGPENDING,
	"RLIMIT_MSGQUEUE":   unix.RLIMIT_MSGQUEUE,
	"RLIMIT_NICE":       unix.RLIMIT_NICE,
	"RLIMIT_RTPRIO":     unix.RLIMIT_RTPRIO,
	"RLIMIT_RTTIME":     unix.RLIMIT_RTTIME,
}

func strToRlimit(key string) (int, error) {
	rl, ok := rlimitMap[key]
	if !ok {
		return 0, fmt.Errorf("wrong rlimit value: %s", key)
	}
	return rl, nil
}

// setupIO modifies the given process config according to the options.
func setupIO(process *libProcess, rootuid, rootgid int, createTTY, detach bool, sockpath string) (*tty, error) {
	if createTTY {
		process.Stdin = nil
		process.Stdout = nil
		process.Stderr = nil
		t := &tty{}
		if !detach {
			if err := t.initHostConsole(); err != nil {
				return nil, err
			}
			parent, child, err := NewSockPair("console")
			if err != nil {
				return nil, err
			}
			process.ConsoleSocket = child
			t.postStart = append(t.postStart, parent, child)
			t.consoleC = make(chan error, 1)
			go func() {
				t.consoleC <- t.recvtty(process, parent)
			}()
		} else {
			// the caller of runc will handle receiving the console master
			conn, err := net.Dial("unix", sockpath)
			if err != nil {
				return nil, err
			}
			uc, ok := conn.(*net.UnixConn)
			if !ok {
				return nil, errors.New("casting to UnixConn failed")
			}
			t.postStart = append(t.postStart, uc)
			socket, err := uc.File()
			if err != nil {
				return nil, err
			}
			t.postStart = append(t.postStart, socket)
			process.ConsoleSocket = socket
		}
		return t, nil
	}
	// when runc will detach the caller provides the stdio to runc via runc's 0,1,2
	// and the container's process inherits runc's stdio.
	if detach {
		if err := inheritStdio(process); err != nil {
			return nil, err
		}
		return &tty{}, nil
	}
	return setupProcessPipes(process, rootuid, rootgid)
}

// NewSockPair returns a new unix socket pair
func NewSockPair(name string) (parent *os.File, child *os.File, err error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), name+"-p"), os.NewFile(uintptr(fds[0]), name+"-c"), nil
}
