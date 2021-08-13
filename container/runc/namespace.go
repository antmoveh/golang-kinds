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
)

type NamespaceType string
type Namespaces []Namespace

// Namespace defines configuration for each namespace.  It specifies an
// alternate path that is able to be joined via setns.
type Namespace struct {
	Type NamespaceType `json:"type"`
	Path string        `json:"path"`
}

const (
	NEWNET    NamespaceType = "NEWNET"
	NEWPID    NamespaceType = "NEWPID"
	NEWNS     NamespaceType = "NEWNS"
	NEWUTS    NamespaceType = "NEWUTS"
	NEWIPC    NamespaceType = "NEWIPC"
	NEWUSER   NamespaceType = "NEWUSER"
	NEWCGROUP NamespaceType = "NEWCGROUP"
)

func (n *Namespaces) index(t NamespaceType) int {
	for i, ns := range *n {
		if ns.Type == t {
			return i
		}
	}
	return -1
}

func (n *Namespaces) Contains(t NamespaceType) bool {
	return n.index(t) != -1
}

func (n *Namespace) GetPath(pid int) string {
	return fmt.Sprintf("/proc/%d/ns/%s", pid, NsName(n.Type))
}

func (n *Namespaces) Remove(t NamespaceType) bool {
	i := n.index(t)
	if i == -1 {
		return false
	}
	*n = append((*n)[:i], (*n)[i+1:]...)
	return true
}

func (n *Namespaces) Add(t NamespaceType, path string) {
	i := n.index(t)
	if i == -1 {
		*n = append(*n, Namespace{Type: t, Path: path})
		return
	}
	(*n)[i].Path = path
}

func (n *Namespaces) PathOf(t NamespaceType) string {
	i := n.index(t)
	if i == -1 {
		return ""
	}
	return (*n)[i].Path
}

// NsName converts the namespace type to its filename
func NsName(ns NamespaceType) string {
	switch ns {
	case NEWNET:
		return "net"
	case NEWNS:
		return "mnt"
	case NEWPID:
		return "pid"
	case NEWIPC:
		return "ipc"
	case NEWUSER:
		return "user"
	case NEWUTS:
		return "uts"
	case NEWCGROUP:
		return "cgroup"
	}
	return ""
}

func setupUserNamespace(spec *Spec, config *Config) error {
	create := func(m LinuxIDMapping) IDMap {
		return IDMap{
			HostID:      int(m.HostID),
			ContainerID: int(m.ContainerID),
			Size:        int(m.Size),
		}
	}
	if spec.Linux != nil {
		for _, m := range spec.Linux.UIDMappings {
			config.UidMappings = append(config.UidMappings, create(m))
		}
		for _, m := range spec.Linux.GIDMappings {
			config.GidMappings = append(config.GidMappings, create(m))
		}
	}
	rootUID, err := config.HostRootUID()
	if err != nil {
		return err
	}
	rootGID, err := config.HostRootGID()
	if err != nil {
		return err
	}
	for _, node := range config.Devices {
		node.Uid = uint32(rootUID)
		node.Gid = uint32(rootGID)
	}
	return nil
}

type IDMap struct {
	ContainerID int `json:"container_id"`
	HostID      int `json:"host_id"`
	Size        int `json:"size"`
}

var (
	errNoUIDMap   = errors.New("User namespaces enabled, but no uid mappings found.")
	errNoUserMap  = errors.New("User namespaces enabled, but no user mapping found.")
	errNoGIDMap   = errors.New("User namespaces enabled, but no gid mappings found.")
	errNoGroupMap = errors.New("User namespaces enabled, but no group mapping found.")
)

// HostUID gets the translated uid for the process on host which could be
// different when user namespaces are enabled.
func (c Config) HostUID(containerId int) (int, error) {
	if c.Namespaces.Contains(NEWUSER) {
		if c.UidMappings == nil {
			return -1, errNoUIDMap
		}
		id, found := c.hostIDFromMapping(containerId, c.UidMappings)
		if !found {
			return -1, errNoUserMap
		}
		return id, nil
	}
	// Return unchanged id.
	return containerId, nil
}

// HostRootUID gets the root uid for the process on host which could be non-zero
// when user namespaces are enabled.
func (c Config) HostRootUID() (int, error) {
	return c.HostUID(0)
}

// HostGID gets the translated gid for the process on host which could be
// different when user namespaces are enabled.
func (c Config) HostGID(containerId int) (int, error) {
	if c.Namespaces.Contains(NEWUSER) {
		if c.GidMappings == nil {
			return -1, errNoGIDMap
		}
		id, found := c.hostIDFromMapping(containerId, c.GidMappings)
		if !found {
			return -1, errNoGroupMap
		}
		return id, nil
	}
	// Return unchanged id.
	return containerId, nil
}

// HostRootGID gets the root gid for the process on host which could be non-zero
// when user namespaces are enabled.
func (c Config) HostRootGID() (int, error) {
	return c.HostGID(0)
}

// Utility function that gets a host ID for a container ID from user namespace map
// if that ID is present in the map.
func (c Config) hostIDFromMapping(containerID int, uMap []IDMap) (int, bool) {
	for _, m := range uMap {
		if (containerID >= m.ContainerID) && (containerID <= (m.ContainerID + m.Size - 1)) {
			hostID := m.HostID + (containerID - m.ContainerID)
			return hostID, true
		}
	}
	return -1, false
}
