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

const (
	// EXT_COPYUP is a directive to copy up the contents of a directory when
	// a tmpfs is mounted over it.
	EXT_COPYUP = 1 << iota //nolint:golint // ignore "don't use ALL_CAPS" warning
)

type ConfigMount struct {
	// Source path for the mount.
	Source string `json:"source"`

	// Destination path for the mount inside the container.
	Destination string `json:"destination"`

	// Device the mount is for.
	Device string `json:"device"`

	// Mount flags.
	Flags int `json:"flags"`

	// Propagation Flags
	PropagationFlags []int `json:"propagation_flags"`

	// Mount data applied to the mount.
	Data string `json:"data"`

	// Relabel source if set, "z" indicates shared, "Z" indicates unshared.
	Relabel string `json:"relabel"`

	// Extensions are additional flags that are specific to runc.
	Extensions int `json:"extensions"`

	// // Optional Command to be run before Source is mounted.
	// PremountCmds []Command `json:"premount_cmds"`
	//
	// // Optional Command to be run after Source is mounted.
	// PostmountCmds []Command `json:"postmount_cmds"`
}

// Config defines configuration options for executing a process inside a contained environment.
type Config struct {
	// NoPivotRoot will use MS_MOVE and a chroot to jail the process into the container's rootfs
	// This is a common option when the container is running in ramdisk
	NoPivotRoot bool `json:"no_pivot_root"`

	// ParentDeathSignal specifies the signal that is sent to the container's process in the case
	// that the parent process dies.
	ParentDeathSignal int `json:"parent_death_signal"`

	// Path to a directory containing the container's root filesystem.
	Rootfs string `json:"rootfs"`

	// Umask is the umask to use inside of the container.
	Umask *uint32 `json:"umask"`

	// Readonlyfs will remount the container's rootfs as readonly where only externally mounted
	// bind mounts are writtable.
	Readonlyfs bool `json:"readonlyfs"`

	// Specifies the mount propagation flags to be applied to /.
	RootPropagation int `json:"rootPropagation"`

	// Mounts specify additional source and destination paths that will be mounted inside the container's
	// rootfs and mount namespace if specified
	Mounts []*ConfigMount `json:"mounts"`

	// The device nodes that should be automatically created within the container upon container start.  Note, make sure that the node is marked as allowed in the cgroup as well!
	Devices []*Device `json:"devices"`

	MountLabel string `json:"mount_label"`

	// Hostname optionally sets the container's hostname if provided
	Hostname string `json:"hostname"`

	// Namespaces specifies the container's namespaces that it should setup when cloning the init process
	// If a namespace is not provided that namespace is shared from the container's parent process
	Namespaces Namespaces `json:"namespaces"`

	// Capabilities specify the capabilities to keep when executing the process inside the container
	// All capabilities not specified will be dropped from the processes capability mask
	Capabilities *Capabilities `json:"capabilities"`

	// Networks specifies the container's network setup to be created
	Networks []*Network `json:"networks"`

	// Routes can be specified to create entries in the route table as the container is started
	// Routes []*Route `json:"routes"`

	// Cgroups specifies specific cgroup settings for the various subsystems that the container is
	// placed into to limit the resources the container has available
	Cgroups *Cgroup `json:"cgroups"`

	// AppArmorProfile specifies the profile to apply to the process running in the container and is
	// change at the time the process is execed
	AppArmorProfile string `json:"apparmor_profile,omitempty"`

	// ProcessLabel specifies the label to apply to the process running in the container.  It is
	// commonly used by selinux
	ProcessLabel string `json:"process_label,omitempty"`

	// Rlimits specifies the resource limits, such as max open files, to set in the container
	// If Rlimits are not set, the container will inherit rlimits from the parent process
	// Rlimits []Rlimit `json:"rlimits,omitempty"`

	// OomScoreAdj specifies the adjustment to be made by the kernel when calculating oom scores
	// for a process. Valid values are between the range [-1000, '1000'], where processes with
	// higher scores are preferred for being killed. If it is unset then we don't touch the current
	// value.
	// More information about kernel oom score calculation here: https://lwn.net/Articles/317814/
	OomScoreAdj *int `json:"oom_score_adj,omitempty"`

	// UidMappings is an array of User ID mappings for User Namespaces
	UidMappings []IDMap `json:"uid_mappings"`

	// GidMappings is an array of Group ID mappings for User Namespaces
	GidMappings []IDMap `json:"gid_mappings"`

	// MaskPaths specifies paths within the container's rootfs to mask over with a bind
	// mount pointing to /dev/null as to prevent reads of the file.
	MaskPaths []string `json:"mask_paths"`

	// ReadonlyPaths specifies paths within the container's rootfs to remount as read-only
	// so that these files prevent any writes.
	ReadonlyPaths []string `json:"readonly_paths"`

	// Sysctl is a map of properties and their values. It is the equivalent of using
	// sysctl -w my.property.name value in Linux.
	Sysctl map[string]string `json:"sysctl"`

	// Seccomp allows actions to be taken whenever a syscall is made within the container.
	// A number of rules are given, each having an action to be taken if a syscall matches it.
	// A default action to be taken if no rules match is also given.
	Seccomp *Seccomp `json:"seccomp"`

	// NoNewPrivileges controls whether processes in the container can gain additional privileges.
	NoNewPrivileges bool `json:"no_new_privileges,omitempty"`

	// Hooks are a collection of actions to perform at various container lifecycle events.
	// CommandHooks are serialized to JSON, but other hooks are not.
	Hooks Hooks

	// Version is the version of opencontainer specification that is supported.
	Version string `json:"version"`

	// Labels are user defined metadata that is stored in the config and populated on the state
	Labels []string `json:"labels"`

	// NoNewKeyring will not allocated a new session keyring for the container.  It will use the
	// callers keyring in this case.
	NoNewKeyring bool `json:"no_new_keyring"`

	// IntelRdt specifies settings for Intel RDT group that the container is placed into
	// to limit the resources (e.g., L3 cache, memory bandwidth) the container has available
	IntelRdt *IntelRdt `json:"intel_rdt,omitempty"`

	// RootlessEUID is set when the runc was launched with non-zero EUID.
	// Note that RootlessEUID is set to false when launched with EUID=0 in userns.
	// When RootlessEUID is set, runc creates a new userns for the container.
	// (config.json needs to contain userns settings)
	RootlessEUID bool `json:"rootless_euid,omitempty"`

	// RootlessCgroups is set when unlikely to have the full access to cgroups.
	// When RootlessCgroups is set, cgroups errors are ignored.
	RootlessCgroups bool `json:"rootless_cgroups,omitempty"`
}

type IntelRdt struct {
	// The schema for L3 cache id and capacity bitmask (CBM)
	// Format: "L3:<cache_id0>=<cbm0>;<cache_id1>=<cbm1>;..."
	L3CacheSchema string `json:"l3_cache_schema,omitempty"`

	// The schema of memory bandwidth per L3 cache id
	// Format: "MB:<cache_id0>=bandwidth0;<cache_id1>=bandwidth1;..."
	// The unit of memory bandwidth is specified in "percentages" by
	// default, and in "MBps" if MBA Software Controller is enabled.
	MemBwSchema string `json:"memBwSchema,omitempty"`
}
