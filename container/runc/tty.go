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
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/signal"
	"sync"

	"github.com/containerd/console"
)

type tty struct {
	epoller     *console.Epoller
	console     *console.EpollConsole
	hostConsole console.Console
	closers     []io.Closer
	postStart   []io.Closer
	wg          sync.WaitGroup
	consoleC    chan error
}

func (t *tty) copyIO(w io.Writer, r io.ReadCloser) {
	defer t.wg.Done()
	_, _ = io.Copy(w, r)
	_ = r.Close()
}

// setup pipes for the process so that advanced features like c/r are able to easily checkpoint
// and restore the process's IO without depending on a host specific path or device
func setupProcessPipes(p *libProcess, rootuid, rootgid int) (*tty, error) {
	i, err := p.InitializeIO(rootuid, rootgid)
	if err != nil {
		return nil, err
	}
	t := &tty{
		closers: []io.Closer{
			i.Stdin,
			i.Stdout,
			i.Stderr,
		},
	}
	// add the process's io to the post start closers if they support close
	for _, cc := range []interface{}{
		p.Stdin,
		p.Stdout,
		p.Stderr,
	} {
		if c, ok := cc.(io.Closer); ok {
			t.postStart = append(t.postStart, c)
		}
	}
	go func() {
		_, _ = io.Copy(i.Stdin, os.Stdin)
		_ = i.Stdin.Close()
	}()
	t.wg.Add(2)
	go t.copyIO(os.Stdout, i.Stdout)
	go t.copyIO(os.Stderr, i.Stderr)
	return t, nil
}

func inheritStdio(process *libProcess) error {
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	return nil
}

func (t *tty) initHostConsole() error {
	// Usually all three (stdin, stdout, and stderr) streams are open to
	// the terminal, but they might be redirected, so try them all.
	for _, s := range []*os.File{os.Stderr, os.Stdout, os.Stdin} {
		c, err := console.ConsoleFromFile(s)
		if err == nil {
			t.hostConsole = c
			return nil
		}
		if errors.Is(err, console.ErrNotAConsole) {
			continue
		}
		// should not happen
		return fmt.Errorf("unable to get console: %w", err)
	}
	// If all streams are redirected, but we still have a controlling
	// terminal, it can be obtained by opening /dev/tty.
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return err
	}
	c, err := console.ConsoleFromFile(tty)
	if err != nil {
		return fmt.Errorf("unable to get console: %w", err)
	}

	t.hostConsole = c
	return nil
}

func (t *tty) recvtty(process *libProcess, socket *os.File) (Err error) {
	f, err := RecvFd(socket)
	if err != nil {
		return err
	}
	cons, err := console.ConsoleFromFile(f)
	if err != nil {
		return err
	}
	err = console.ClearONLCR(cons.Fd())
	if err != nil {
		return err
	}
	epoller, err := console.NewEpoller()
	if err != nil {
		return err
	}
	epollConsole, err := epoller.Add(cons)
	if err != nil {
		return err
	}
	defer func() {
		if Err != nil {
			_ = epollConsole.Close()
		}
	}()
	go func() { _ = epoller.Wait() }()
	go func() { _, _ = io.Copy(epollConsole, os.Stdin) }()
	t.wg.Add(1)
	go t.copyIO(os.Stdout, epollConsole)

	// Set raw mode for the controlling terminal.
	if err := t.hostConsole.SetRaw(); err != nil {
		return fmt.Errorf("failed to set the terminal from the stdin: %w", err)
	}
	go handleInterrupt(t.hostConsole)

	t.epoller = epoller
	t.console = epollConsole
	t.closers = []io.Closer{epollConsole}
	return nil
}

func handleInterrupt(c console.Console) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	_ = c.Reset()
	os.Exit(0)
}

func (t *tty) waitConsole() error {
	if t.consoleC != nil {
		return <-t.consoleC
	}
	return nil
}

// ClosePostStart closes any fds that are provided to the container and dup2'd
// so that we no longer have copy in our process.
func (t *tty) ClosePostStart() error {
	for _, c := range t.postStart {
		_ = c.Close()
	}
	return nil
}

// Close closes all open fds for the tty and/or restores the original
// stdin state to what it was prior to the container execution
func (t *tty) Close() error {
	// ensure that our side of the fds are always closed
	for _, c := range t.postStart {
		_ = c.Close()
	}
	// the process is gone at this point, shutting down the console if we have
	// one and wait for all IO to be finished
	if t.console != nil && t.epoller != nil {
		_ = t.console.Shutdown(t.epoller.CloseConsole)
	}
	t.wg.Wait()
	for _, c := range t.closers {
		_ = c.Close()
	}
	if t.hostConsole != nil {
		_ = t.hostConsole.Reset()
	}
	return nil
}

func (t *tty) resize() error {
	if t.console == nil || t.hostConsole == nil {
		return nil
	}
	return t.console.ResizeFrom(t.hostConsole)
}

func (p *libProcess) InitializeIO(rootuid, rootgid int) (i *IO, err error) {
	var fds []uintptr
	i = &IO{}
	// cleanup in case of an error
	defer func() {
		if err != nil {
			for _, fd := range fds {
				_ = unix.Close(int(fd))
			}
		}
	}()
	// STDIN
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	fds = append(fds, r.Fd(), w.Fd())
	p.Stdin, i.Stdin = r, w
	// STDOUT
	if r, w, err = os.Pipe(); err != nil {
		return nil, err
	}
	fds = append(fds, r.Fd(), w.Fd())
	p.Stdout, i.Stdout = w, r
	// STDERR
	if r, w, err = os.Pipe(); err != nil {
		return nil, err
	}
	fds = append(fds, r.Fd(), w.Fd())
	p.Stderr, i.Stderr = w, r
	// change ownership of the pipes in case we are in a user namespace
	for _, fd := range fds {
		if err := unix.Fchown(int(fd), rootuid, rootgid); err != nil {
			return nil, err
		}
	}
	return i, nil
}

type IO struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

const MaxNameLen = 4096

// oobSpace is the size of the oob slice required to store a single FD. Note
// that unix.UnixRights appears to make the assumption that fd is always int32,
// so sizeof(fd) = 4.
var oobSpace = unix.CmsgSpace(4)

// RecvFd waits for a file descriptor to be sent over the given AF_UNIX
// socket. The file name of the remote file descriptor will be recreated
// locally (it is sent as non-auxiliary data in the same payload).
func RecvFd(socket *os.File) (*os.File, error) {
	// For some reason, unix.Recvmsg uses the length rather than the capacity
	// when passing the msg_controllen and other attributes to recvmsg.  So we
	// have to actually set the length.
	name := make([]byte, MaxNameLen)
	oob := make([]byte, oobSpace)

	sockfd := socket.Fd()
	n, oobn, _, _, err := unix.Recvmsg(int(sockfd), name, oob, 0)
	if err != nil {
		return nil, err
	}

	if n >= MaxNameLen || oobn != oobSpace {
		return nil, fmt.Errorf("recvfd: incorrect number of bytes read (n=%d oobn=%d)", n, oobn)
	}

	// Truncate.
	name = name[:n]
	oob = oob[:oobn]

	scms, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}
	if len(scms) != 1 {
		return nil, fmt.Errorf("recvfd: number of SCMs is not 1: %d", len(scms))
	}
	scm := scms[0]

	fds, err := unix.ParseUnixRights(&scm)
	if err != nil {
		return nil, err
	}
	if len(fds) != 1 {
		return nil, fmt.Errorf("recvfd: number of fds is not 1: %d", len(fds))
	}
	fd := uintptr(fds[0])

	return os.NewFile(fd, string(name)), nil
}

// SendFd sends a file descriptor over the given AF_UNIX socket. In
// addition, the file.Name() of the given file will also be sent as
// non-auxiliary data in the same payload (allowing to send contextual
// information for a file descriptor).
func SendFd(socket *os.File, name string, fd uintptr) error {
	if len(name) >= MaxNameLen {
		return fmt.Errorf("sendfd: filename too long: %s", name)
	}
	oob := unix.UnixRights(int(fd))
	return unix.Sendmsg(int(socket.Fd()), []byte(name), oob, nil, 0)
}
