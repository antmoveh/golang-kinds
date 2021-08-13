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
	"golang.org/x/sys/unix"
	"os"
)

const (
	WildcardDevice Type = 'a'
	BlockDevice    Type = 'b'
	CharDevice     Type = 'c' // or 'u'
	FifoDevice     Type = 'p'

	Wildcard = -1
)

type Device struct {
	Rule

	// Path to the device.
	Path string `json:"path"`

	// FileMode permission bits for the device.
	FileMode os.FileMode `json:"file_mode"`

	// Uid of the device.
	Uid uint32 `json:"uid"`

	// Gid of the device.
	Gid uint32 `json:"gid"`
}

type Permissions string
type Type rune

type Rule struct {
	// Type of device ('c' for char, 'b' for block). If set to 'a', this rule
	// acts as a wildcard and all fields other than Allow are ignored.
	Type Type `json:"type"`

	// Major is the device's major number.
	Major int64 `json:"major"`

	// Minor is the device's minor number.
	Minor int64 `json:"minor"`

	// Permissions is the set of permissions that this rule applies to (in the
	// cgroupv1 format -- any combination of "rwm").
	Permissions Permissions `json:"permissions"`

	// Allow specifies whether this rule is allowed.
	Allow bool `json:"allow"`
}

// NewWeightDevice returns a configured WeightDevice pointer
func NewWeightDevice(major, minor int64, weight, leafWeight uint16) *WeightDevice {
	wd := &WeightDevice{}
	wd.Major = major
	wd.Minor = minor
	wd.Weight = weight
	wd.LeafWeight = leafWeight
	return wd
}

// NewThrottleDevice returns a configured ThrottleDevice pointer
func NewThrottleDevice(major, minor int64, rate uint64) *ThrottleDevice {
	td := &ThrottleDevice{}
	td.Major = major
	td.Minor = minor
	td.Rate = rate
	return td
}

var mountPropagationMapping = map[string]int{
	"rprivate":    unix.MS_PRIVATE | unix.MS_REC,
	"private":     unix.MS_PRIVATE,
	"rslave":      unix.MS_SLAVE | unix.MS_REC,
	"slave":       unix.MS_SLAVE,
	"rshared":     unix.MS_SHARED | unix.MS_REC,
	"shared":      unix.MS_SHARED,
	"runbindable": unix.MS_UNBINDABLE | unix.MS_REC,
	"unbindable":  unix.MS_UNBINDABLE,
	"":            0,
}

var AllowedDevices = []*Device{
	// allow mknod for any device
	{
		Rule: Rule{
			Type:        CharDevice,
			Major:       Wildcard,
			Minor:       Wildcard,
			Permissions: "m",
			Allow:       true,
		},
	},
	{
		Rule: Rule{
			Type:        BlockDevice,
			Major:       Wildcard,
			Minor:       Wildcard,
			Permissions: "m",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/null",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       3,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/random",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       8,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/full",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       7,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/tty",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       5,
			Minor:       0,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/zero",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       5,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/urandom",
		FileMode: 0o666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       9,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	// /dev/pts/ - pts namespaces are "coming soon"
	{
		Rule: Rule{
			Type:        CharDevice,
			Major:       136,
			Minor:       Wildcard,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Rule: Rule{
			Type:        CharDevice,
			Major:       5,
			Minor:       2,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	// tuntap
	{
		Rule: Rule{
			Type:        CharDevice,
			Major:       10,
			Minor:       200,
			Permissions: "rwm",
			Allow:       true,
		},
	},
}
