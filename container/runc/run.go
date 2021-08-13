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
	"fmt"
	"github.com/coreos/go-systemd/v22/activation"
	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var namespaceMapping = map[LinuxNamespaceType]NamespaceType{
	PIDNamespace:     NEWPID,
	NetworkNamespace: NEWNET,
	MountNamespace:   NEWNS,
	UserNamespace:    NEWUSER,
	IPCNamespace:     NEWIPC,
	UTSNamespace:     NEWUTS,
	CgroupNamespace:  NEWCGROUP,
}

func startContainer(ctx context.Context, spec *Spec, criuOpts *CriuOpts) (int, error) {
	id := "9527"

	container, err := createContainer(ctx, id, spec)
	if err != nil {
		return -1, err
	}

	// Support on-demand socket activation by passing file descriptors into the container init process.
	listenFDs := []*os.File{}
	if os.Getenv("LISTEN_FDS") != "" {
		listenFDs = activation.Files(false)
	}

	logLevel := "debug"

	r := &runner{
		enableSubreaper: true,
		shouldDestroy:   true,
		container:       container,
		listenFDs:       listenFDs,
		notifySocket:    &notifySocket{},
		consoleSocket:   "",
		detach:          true,
		pidFile:         "",
		preserveFDs:     1,
		action:          1,
		criuOpts:        criuOpts,
		init:            true,
		logLevel:        logLevel,
	}
	return r.run(spec.Process)
}

func createContainer(context context.Context, id string, spec *Spec) (Container, error) {

	config, err := CreateLibcontainerConfig(&CreateOpts{
		CgroupName:       id,
		UseSystemdCgroup: true,
		NoPivotRoot:      false,
		NoNewKeyring:     false,
		Spec:             spec,
		RootlessEUID:     os.Geteuid() != 0,
		RootlessCgroups:  false,
	})
	if err != nil {
		return nil, err
	}

	factory, err := loadFactory(context)
	if err != nil {
		return nil, err
	}
	return factory.Create(id, config)
}

// CreateLibcontainerConfig creates a new libcontainer configuration from a
// given specification and a cgroup name
func CreateLibcontainerConfig(opts *CreateOpts) (*Config, error) {
	// runc's cwd will always be the bundle path
	rcwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwd, err := filepath.Abs(rcwd)
	if err != nil {
		return nil, err
	}
	spec := opts.Spec
	if spec.Root == nil {
		return nil, errors.New("Root must be specified")
	}
	rootfsPath := spec.Root.Path
	if !filepath.IsAbs(rootfsPath) {
		rootfsPath = filepath.Join(cwd, rootfsPath)
	}
	labels := []string{}
	for k, v := range spec.Annotations {
		labels = append(labels, k+"="+v)
	}
	config := &Config{
		Rootfs:          rootfsPath,
		NoPivotRoot:     opts.NoPivotRoot,
		Readonlyfs:      spec.Root.Readonly,
		Hostname:        spec.Hostname,
		Labels:          append(labels, "bundle="+cwd),
		NoNewKeyring:    opts.NoNewKeyring,
		RootlessEUID:    opts.RootlessEUID,
		RootlessCgroups: opts.RootlessCgroups,
	}

	for _, m := range spec.Mounts {
		cm, err := createLibcontainerMount(cwd, m)
		if err != nil {
			return nil, fmt.Errorf("invalid mount %+v: %w", m, err)
		}
		config.Mounts = append(config.Mounts, cm)
	}

	defaultDevs, err := createDevices(spec, config)
	if err != nil {
		return nil, err
	}

	c, err := CreateCgroupConfig(opts, defaultDevs)
	if err != nil {
		return nil, err
	}

	config.Cgroups = c
	// set linux-specific config
	if spec.Linux != nil {
		var exists bool
		if config.RootPropagation, exists = mountPropagationMapping[spec.Linux.RootfsPropagation]; !exists {
			return nil, fmt.Errorf("rootfsPropagation=%v is not supported", spec.Linux.RootfsPropagation)
		}
		if config.NoPivotRoot && (config.RootPropagation&unix.MS_PRIVATE != 0) {
			return nil, errors.New("rootfsPropagation of [r]private is not safe without pivot_root")
		}

		for _, ns := range spec.Linux.Namespaces {
			t, exists := namespaceMapping[ns.Type]
			if !exists {
				return nil, fmt.Errorf("namespace %q does not exist", ns)
			}
			if config.Namespaces.Contains(t) {
				return nil, fmt.Errorf("malformed spec file: duplicated ns %q", ns)
			}
			config.Namespaces.Add(t, ns.Path)
		}
		if config.Namespaces.Contains(NEWNET) && config.Namespaces.PathOf(NEWNET) == "" {
			config.Networks = []*Network{
				{
					Type: "loopback",
				},
			}
		}
		if config.Namespaces.Contains(NEWUSER) {
			if err := setupUserNamespace(spec, config); err != nil {
				return nil, err
			}
		}
		config.MaskPaths = spec.Linux.MaskedPaths
		config.ReadonlyPaths = spec.Linux.ReadonlyPaths
		config.MountLabel = spec.Linux.MountLabel
		config.Sysctl = spec.Linux.Sysctl
		if spec.Linux.Seccomp != nil {
			seccomp, err := SetupSeccomp(spec.Linux.Seccomp)
			if err != nil {
				return nil, err
			}
			config.Seccomp = seccomp
		}
		if spec.Linux.IntelRdt != nil {
			config.IntelRdt = &IntelRdt{
				L3CacheSchema: spec.Linux.IntelRdt.L3CacheSchema,
				MemBwSchema:   spec.Linux.IntelRdt.MemBwSchema,
			}
		}
	}
	if spec.Process != nil {
		config.OomScoreAdj = spec.Process.OOMScoreAdj
		config.NoNewPrivileges = spec.Process.NoNewPrivileges
		config.Umask = spec.Process.User.Umask
		config.ProcessLabel = spec.Process.SelinuxLabel
		if spec.Process.Capabilities != nil {
			config.Capabilities = &Capabilities{
				Bounding:    spec.Process.Capabilities.Bounding,
				Effective:   spec.Process.Capabilities.Effective,
				Permitted:   spec.Process.Capabilities.Permitted,
				Inheritable: spec.Process.Capabilities.Inheritable,
				Ambient:     spec.Process.Capabilities.Ambient,
			}
		}
	}
	config.Version = "1.0"
	return config, nil
}

func createLibcontainerMount(cwd string, m Mount) (*ConfigMount, error) {
	if !filepath.IsAbs(m.Destination) {
		// Relax validation for backward compatibility
		// TODO (runc v1.x.x): change warning to an error
		// return nil, fmt.Errorf("mount destination %s is not absolute", m.Destination)
		logrus.Warnf("mount destination %s is not absolute. Support for non-absolute mount destinations will be removed in a future release.", m.Destination)
	}
	flags, pgflags, data, ext := parseMountOptions(m.Options)
	source := m.Source
	device := m.Type
	if flags&unix.MS_BIND != 0 {
		// Any "type" the user specified is meaningless (and ignored) for
		// bind-mounts -- so we set it to "bind" because rootfs_linux.go
		// (incorrectly) relies on this for some checks.
		device = "bind"
		if !filepath.IsAbs(source) {
			source = filepath.Join(cwd, m.Source)
		}
	}
	return &ConfigMount{
		Device:           device,
		Source:           source,
		Destination:      m.Destination,
		Data:             data,
		Flags:            flags,
		PropagationFlags: pgflags,
		Extensions:       ext,
	}, nil
}

// parseMountOptions parses the string and returns the flags, propagation
// flags and any mount data that it contains.
func parseMountOptions(options []string) (int, []int, string, int) {
	var (
		flag     int
		pgflag   []int
		data     []string
		extFlags int
	)
	flags := map[string]struct {
		clear bool
		flag  int
	}{
		"acl":           {false, unix.MS_POSIXACL},
		"async":         {true, unix.MS_SYNCHRONOUS},
		"atime":         {true, unix.MS_NOATIME},
		"bind":          {false, unix.MS_BIND},
		"defaults":      {false, 0},
		"dev":           {true, unix.MS_NODEV},
		"diratime":      {true, unix.MS_NODIRATIME},
		"dirsync":       {false, unix.MS_DIRSYNC},
		"exec":          {true, unix.MS_NOEXEC},
		"iversion":      {false, unix.MS_I_VERSION},
		"lazytime":      {false, unix.MS_LAZYTIME},
		"loud":          {true, unix.MS_SILENT},
		"mand":          {false, unix.MS_MANDLOCK},
		"noacl":         {true, unix.MS_POSIXACL},
		"noatime":       {false, unix.MS_NOATIME},
		"nodev":         {false, unix.MS_NODEV},
		"nodiratime":    {false, unix.MS_NODIRATIME},
		"noexec":        {false, unix.MS_NOEXEC},
		"noiversion":    {true, unix.MS_I_VERSION},
		"nolazytime":    {true, unix.MS_LAZYTIME},
		"nomand":        {true, unix.MS_MANDLOCK},
		"norelatime":    {true, unix.MS_RELATIME},
		"nostrictatime": {true, unix.MS_STRICTATIME},
		"nosuid":        {false, unix.MS_NOSUID},
		"rbind":         {false, unix.MS_BIND | unix.MS_REC},
		"relatime":      {false, unix.MS_RELATIME},
		"remount":       {false, unix.MS_REMOUNT},
		"ro":            {false, unix.MS_RDONLY},
		"rw":            {true, unix.MS_RDONLY},
		"silent":        {false, unix.MS_SILENT},
		"strictatime":   {false, unix.MS_STRICTATIME},
		"suid":          {true, unix.MS_NOSUID},
		"sync":          {false, unix.MS_SYNCHRONOUS},
	}
	propagationFlags := map[string]int{
		"private":     unix.MS_PRIVATE,
		"shared":      unix.MS_SHARED,
		"slave":       unix.MS_SLAVE,
		"unbindable":  unix.MS_UNBINDABLE,
		"rprivate":    unix.MS_PRIVATE | unix.MS_REC,
		"rshared":     unix.MS_SHARED | unix.MS_REC,
		"rslave":      unix.MS_SLAVE | unix.MS_REC,
		"runbindable": unix.MS_UNBINDABLE | unix.MS_REC,
	}
	extensionFlags := map[string]struct {
		clear bool
		flag  int
	}{
		"tmpcopyup": {false, EXT_COPYUP},
	}
	for _, o := range options {
		// If the option does not exist in the flags table or the flag
		// is not supported on the platform,
		// then it is a data value for a specific fs type
		if f, exists := flags[o]; exists && f.flag != 0 {
			if f.clear {
				flag &= ^f.flag
			} else {
				flag |= f.flag
			}
		} else if f, exists := propagationFlags[o]; exists && f != 0 {
			pgflag = append(pgflag, f)
		} else if f, exists := extensionFlags[o]; exists && f.flag != 0 {
			if f.clear {
				extFlags &= ^f.flag
			} else {
				extFlags |= f.flag
			}
		} else {
			data = append(data, o)
		}
	}
	return flag, pgflag, strings.Join(data, ","), extFlags
}

func createDevices(spec *Spec, config *Config) ([]*Device, error) {
	// If a spec device is redundant with a default device, remove that default
	// device (the spec one takes priority).
	dedupedAllowDevs := []*Device{}

next:
	for _, ad := range AllowedDevices {
		if ad.Path != "" {
			for _, sd := range spec.Linux.Devices {
				if sd.Path == ad.Path {
					continue next
				}
			}
		}
		dedupedAllowDevs = append(dedupedAllowDevs, ad)
		if ad.Path != "" {
			config.Devices = append(config.Devices, ad)
		}
	}

	// Merge in additional devices from the spec.
	if spec.Linux != nil {
		for _, d := range spec.Linux.Devices {
			var uid, gid uint32
			var filemode os.FileMode = 0o666

			if d.UID != nil {
				uid = *d.UID
			}
			if d.GID != nil {
				gid = *d.GID
			}
			dt, err := stringToDeviceRune(d.Type)
			if err != nil {
				return nil, err
			}
			if d.FileMode != nil {
				filemode = *d.FileMode &^ unix.S_IFMT
			}
			device := &Device{
				Rule: Rule{
					Type:  dt,
					Major: d.Major,
					Minor: d.Minor,
				},
				Path:     d.Path,
				FileMode: filemode,
				Uid:      uid,
				Gid:      gid,
			}
			config.Devices = append(config.Devices, device)
		}
	}

	return dedupedAllowDevs, nil
}

func stringToDeviceRune(s string) (Type, error) {
	switch s {
	case "p":
		return FifoDevice, nil
	case "u", "c":
		return CharDevice, nil
	case "b":
		return BlockDevice, nil
	default:
		return 0, fmt.Errorf("invalid device type %q", s)
	}
}

func CreateCgroupConfig(opts *CreateOpts, defaultDevs []*Device) (*Cgroup, error) {
	var (
		myCgroupPath string

		spec             = opts.Spec
		useSystemdCgroup = opts.UseSystemdCgroup
		name             = opts.CgroupName
	)

	c := &Cgroup{
		Resources: &Resources{},
	}

	if useSystemdCgroup {
		sp, err := initSystemdProps(spec)
		if err != nil {
			return nil, err
		}
		c.SystemdProps = sp
	}

	if spec.Linux != nil && spec.Linux.CgroupsPath != "" {
		if useSystemdCgroup {
			myCgroupPath = spec.Linux.CgroupsPath
		} else {
			myCgroupPath = CleanPath(spec.Linux.CgroupsPath)
		}
	}

	if useSystemdCgroup {
		if myCgroupPath == "" {
			// Default for c.Parent is set by systemd cgroup drivers.
			c.ScopePrefix = "runc"
			c.Name = name
		} else {
			// Parse the path from expected "slice:prefix:name"
			// for e.g. "system.slice:docker:1234"
			parts := strings.Split(myCgroupPath, ":")
			if len(parts) != 3 {
				return nil, fmt.Errorf("expected cgroupsPath to be of format \"slice:prefix:name\" for systemd cgroups, got %q instead", myCgroupPath)
			}
			c.Parent = parts[0]
			c.ScopePrefix = parts[1]
			c.Name = parts[2]
		}
	} else {
		if myCgroupPath == "" {
			c.Name = name
		}
		c.Path = myCgroupPath
	}

	// In rootless containers, any attempt to make cgroup changes is likely to fail.
	// libcontainer will validate this but ignores the error.
	if spec.Linux != nil {
		r := spec.Linux.Resources
		if r != nil {
			for i, d := range spec.Linux.Resources.Devices {
				var (
					t     = "a"
					major = int64(-1)
					minor = int64(-1)
				)
				if d.Type != "" {
					t = d.Type
				}
				if d.Major != nil {
					major = *d.Major
				}
				if d.Minor != nil {
					minor = *d.Minor
				}
				if d.Access == "" {
					return nil, fmt.Errorf("device access at %d field cannot be empty", i)
				}
				dt, err := stringToCgroupDeviceRune(t)
				if err != nil {
					return nil, err
				}
				c.Resources.Devices = append(c.Resources.Devices, &Rule{
					Type:        dt,
					Major:       major,
					Minor:       minor,
					Permissions: Permissions(d.Access),
					Allow:       d.Allow,
				})
			}
			if r.Memory != nil {
				if r.Memory.Limit != nil {
					c.Resources.Memory = *r.Memory.Limit
				}
				if r.Memory.Reservation != nil {
					c.Resources.MemoryReservation = *r.Memory.Reservation
				}
				if r.Memory.Swap != nil {
					c.Resources.MemorySwap = *r.Memory.Swap
				}
				if r.Memory.Kernel != nil || r.Memory.KernelTCP != nil {
					logrus.Warn("Kernel memory settings are ignored and will be removed")
				}
				if r.Memory.Swappiness != nil {
					c.Resources.MemorySwappiness = r.Memory.Swappiness
				}
				if r.Memory.DisableOOMKiller != nil {
					c.Resources.OomKillDisable = *r.Memory.DisableOOMKiller
				}
			}
			if r.CPU != nil {
				if r.CPU.Shares != nil {
					c.Resources.CpuShares = *r.CPU.Shares

					// CpuWeight is used for cgroupv2 and should be converted
					c.Resources.CpuWeight = ConvertCPUSharesToCgroupV2Value(c.Resources.CpuShares)
				}
				if r.CPU.Quota != nil {
					c.Resources.CpuQuota = *r.CPU.Quota
				}
				if r.CPU.Period != nil {
					c.Resources.CpuPeriod = *r.CPU.Period
				}
				if r.CPU.RealtimeRuntime != nil {
					c.Resources.CpuRtRuntime = *r.CPU.RealtimeRuntime
				}
				if r.CPU.RealtimePeriod != nil {
					c.Resources.CpuRtPeriod = *r.CPU.RealtimePeriod
				}
				c.Resources.CpusetCpus = r.CPU.Cpus
				c.Resources.CpusetMems = r.CPU.Mems
			}
			if r.Pids != nil {
				c.Resources.PidsLimit = r.Pids.Limit
			}
			if r.BlockIO != nil {
				if r.BlockIO.Weight != nil {
					c.Resources.BlkioWeight = *r.BlockIO.Weight
				}
				if r.BlockIO.LeafWeight != nil {
					c.Resources.BlkioLeafWeight = *r.BlockIO.LeafWeight
				}
				if r.BlockIO.WeightDevice != nil {
					for _, wd := range r.BlockIO.WeightDevice {
						var weight, leafWeight uint16
						if wd.Weight != nil {
							weight = *wd.Weight
						}
						if wd.LeafWeight != nil {
							leafWeight = *wd.LeafWeight
						}
						weightDevice := NewWeightDevice(wd.Major, wd.Minor, weight, leafWeight)
						c.Resources.BlkioWeightDevice = append(c.Resources.BlkioWeightDevice, weightDevice)
					}
				}
				if r.BlockIO.ThrottleReadBpsDevice != nil {
					for _, td := range r.BlockIO.ThrottleReadBpsDevice {
						rate := td.Rate
						throttleDevice := NewThrottleDevice(td.Major, td.Minor, rate)
						c.Resources.BlkioThrottleReadBpsDevice = append(c.Resources.BlkioThrottleReadBpsDevice, throttleDevice)
					}
				}
				if r.BlockIO.ThrottleWriteBpsDevice != nil {
					for _, td := range r.BlockIO.ThrottleWriteBpsDevice {
						rate := td.Rate
						throttleDevice := NewThrottleDevice(td.Major, td.Minor, rate)
						c.Resources.BlkioThrottleWriteBpsDevice = append(c.Resources.BlkioThrottleWriteBpsDevice, throttleDevice)
					}
				}
				if r.BlockIO.ThrottleReadIOPSDevice != nil {
					for _, td := range r.BlockIO.ThrottleReadIOPSDevice {
						rate := td.Rate
						throttleDevice := NewThrottleDevice(td.Major, td.Minor, rate)
						c.Resources.BlkioThrottleReadIOPSDevice = append(c.Resources.BlkioThrottleReadIOPSDevice, throttleDevice)
					}
				}
				if r.BlockIO.ThrottleWriteIOPSDevice != nil {
					for _, td := range r.BlockIO.ThrottleWriteIOPSDevice {
						rate := td.Rate
						throttleDevice := NewThrottleDevice(td.Major, td.Minor, rate)
						c.Resources.BlkioThrottleWriteIOPSDevice = append(c.Resources.BlkioThrottleWriteIOPSDevice, throttleDevice)
					}
				}
			}
			for _, l := range r.HugepageLimits {
				c.Resources.HugetlbLimit = append(c.Resources.HugetlbLimit, &HugepageLimit{
					Pagesize: l.Pagesize,
					Limit:    l.Limit,
				})
			}
			if r.Network != nil {
				if r.Network.ClassID != nil {
					c.Resources.NetClsClassid = *r.Network.ClassID
				}
				for _, m := range r.Network.Priorities {
					c.Resources.NetPrioIfpriomap = append(c.Resources.NetPrioIfpriomap, &IfPrioMap{
						Interface: m.Name,
						Priority:  int64(m.Priority),
					})
				}
			}
			if len(r.Unified) > 0 {
				// copy the map
				c.Resources.Unified = make(map[string]string, len(r.Unified))
				for k, v := range r.Unified {
					c.Resources.Unified[k] = v
				}
			}
		}
	}

	// Append the default allowed devices to the end of the list.
	for _, device := range defaultDevs {
		c.Resources.Devices = append(c.Resources.Devices, &device.Rule)
	}
	return c, nil
}

var isValidName = regexp.MustCompile(`^[a-zA-Z]{3,}$`).MatchString
var isSecSuffix = regexp.MustCompile(`[a-z]Sec$`).MatchString

// Some systemd properties are documented as having "Sec" suffix
// (e.g. TimeoutStopSec) but are expected to have "USec" suffix
// here, so let's provide conversion to improve compatibility.
func convertSecToUSec(value dbus.Variant) (dbus.Variant, error) {
	var sec uint64
	const M = 1000000
	vi := value.Value()
	switch value.Signature().String() {
	case "y":
		sec = uint64(vi.(byte)) * M
	case "n":
		sec = uint64(vi.(int16)) * M
	case "q":
		sec = uint64(vi.(uint16)) * M
	case "i":
		sec = uint64(vi.(int32)) * M
	case "u":
		sec = uint64(vi.(uint32)) * M
	case "x":
		sec = uint64(vi.(int64)) * M
	case "t":
		sec = vi.(uint64) * M
	case "d":
		sec = uint64(vi.(float64) * M)
	default:
		return value, errors.New("not a number")
	}
	return dbus.MakeVariant(sec), nil
}

func initSystemdProps(spec *Spec) ([]systemdDbus.Property, error) {
	const keyPrefix = "org.systemd.property."
	var sp []systemdDbus.Property

	for k, v := range spec.Annotations {
		name := strings.TrimPrefix(k, keyPrefix)
		if len(name) == len(k) { // prefix not there
			continue
		}
		if !isValidName(name) {
			return nil, fmt.Errorf("Annotation %s name incorrect: %s", k, name)
		}
		value, err := dbus.ParseVariant(v, dbus.Signature{})
		if err != nil {
			return nil, fmt.Errorf("Annotation %s=%s value parse error: %w", k, v, err)
		}
		if isSecSuffix(name) {
			name = strings.TrimSuffix(name, "Sec") + "USec"
			value, err = convertSecToUSec(value)
			if err != nil {
				return nil, fmt.Errorf("Annotation %s=%s value parse error: %w", k, v, err)
			}
		}
		sp = append(sp, systemdDbus.Property{Name: name, Value: value})
	}

	return sp, nil
}

func stringToCgroupDeviceRune(s string) (Type, error) {
	switch s {
	case "a":
		return WildcardDevice, nil
	case "b":
		return BlockDevice, nil
	case "c":
		return CharDevice, nil
	default:
		return 0, fmt.Errorf("invalid cgroup device type %q", s)
	}
}

func CleanPath(path string) string {
	// Deal with empty strings nicely.
	if path == "" {
		return ""
	}

	// Ensure that all paths are cleaned (especially problematic ones like
	// "/../../../../../" which can cause lots of issues).
	path = filepath.Clean(path)

	// If the path isn't absolute, we need to do more processing to fix paths
	// such as "../../../../<etc>/some/path". We also shouldn't convert absolute
	// paths to relative ones.
	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		// This can't fail, as (by definition) all paths are relative to root.
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}

	// Clean the path again for good measure.
	return filepath.Clean(path)
}

func ConvertCPUSharesToCgroupV2Value(cpuShares uint64) uint64 {
	if cpuShares == 0 {
		return 0
	}
	return (1 + ((cpuShares-2)*9999)/262142)
}
