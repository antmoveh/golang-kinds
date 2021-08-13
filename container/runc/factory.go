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
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
)

func loadFactory(ctx context.Context) (Factory, error) {
	root := "root"
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// We default to cgroupfs, and can only use systemd if the system is a
	// systemd box.
	cgroupManager := Cgroupfs

	cgroupManager = SystemdCgroups

	intelRdtManager := libcontainer.IntelRdtFs

	// We resolve the paths for {newuidmap,newgidmap} from the context of runc,
	// to avoid doing a path lookup in the nsexec context. TODO: The binary
	// names are not currently configurable.
	newuidmap, err := exec.LookPath("newuidmap")
	if err != nil {
		newuidmap = ""
	}
	newgidmap, err := exec.LookPath("newgidmap")
	if err != nil {
		newgidmap = ""
	}

	return libcontainer.New(abs, cgroupManager, intelRdtManager,
		libcontainer.CriuPath(context.GlobalString("criu")),
		libcontainer.NewuidmapPath(newuidmap),
		libcontainer.NewgidmapPath(newgidmap))
}

func cgroupfs(l *LinuxFactory, rootless bool) error {
	if cgroups.IsCgroup2UnifiedMode() {
		return cgroupfs2(l, rootless)
	}
	l.NewCgroupsManager = func(config *configs.Cgroup, paths map[string]string) cgroups.Manager {
		return fs.NewManager(config, paths, rootless)
	}
	return nil
}

// Cgroupfs is an options func to configure a LinuxFactory to return containers
// that use the native cgroups filesystem implementation to create and manage
// cgroups.
func Cgroupfs(l *LinuxFactory) error {
	return cgroupfs(l, false)
}

// LinuxFactory implements the default factory interface for linux based systems.
type LinuxFactory struct {
	// Root directory for the factory to store state.
	Root string

	// InitPath is the path for calling the init responsibilities for spawning
	// a container.
	InitPath string

	// InitArgs are arguments for calling the init responsibilities for spawning
	// a container.
	InitArgs []string

	// CriuPath is the path to the criu binary used for checkpoint and restore of
	// containers.
	CriuPath string

	// New{u,g}idmapPath is the path to the binaries used for mapping with
	// rootless containers.
	NewuidmapPath string
	NewgidmapPath string

	// Validator provides validation to container configurations.
	Validator validate.Validator

	// NewCgroupsManager returns an initialized cgroups manager for a single container.
	NewCgroupsManager func(config *Cgroup, paths map[string]string) Manager

	// NewIntelRdtManager returns an initialized Intel RDT manager for a single container.
	NewIntelRdtManager func(config *Config, id string, path string) Manager
}

// SystemdCgroups is an options func to configure a LinuxFactory to return
// containers that use systemd to create and manage cgroups.
func SystemdCgroups(l *LinuxFactory) error {

	l.NewCgroupsManager = func(config *Cgroup, paths map[string]string) Manager {
		return systemd.NewLegacyManager(config, paths)
	}

	return nil
}

func IntelRdtFs(l *LinuxFactory) error {
	if !intelrdt.IsCATEnabled() && !intelrdt.IsMBAEnabled() {
		l.NewIntelRdtManager = nil
	} else {
		l.NewIntelRdtManager = func(config *Config, id string, path string) .Manager {
			return intelrdt.NewManager(config, id, path)
		}
	}
	return nil
}

const (
	stateFilename    = "state.json"
	execFifoFilename = "exec.fifo"
)

var (
	idRegex      = regexp.MustCompile(`^[\w+-\.]+$`)
	errNoSystemd = errors.New("systemd not running on this host, can't use systemd as cgroups manager")
)
