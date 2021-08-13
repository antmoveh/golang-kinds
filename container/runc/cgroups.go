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

type Manager interface {
	// Apply creates a cgroup, if not yet created, and adds a process
	// with the specified pid into that cgroup.  A special value of -1
	// can be used to merely create a cgroup.
	Apply(pid int) error

	// GetPids returns the PIDs of all processes inside the cgroup.
	GetPids() ([]int, error)

	// GetAllPids returns the PIDs of all processes inside the cgroup
	// any all its sub-cgroups.
	GetAllPids() ([]int, error)

	// GetStats returns cgroups statistics.
	// GetStats() (*Stats, error)

	// Freeze sets the freezer cgroup to the specified state.
	Freeze(state FreezerState) error

	// Destroy removes cgroup.
	Destroy() error

	// Path returns a cgroup path to the specified controller/subsystem.
	// For cgroupv2, the argument is unused and can be empty.
	Path(string) string

	// Set sets cgroup resources parameters/limits. If the argument is nil,
	// the resources specified during Manager creation (or the previous call
	// to Set) are used.
	Set(r *Resources) error

	// GetPaths returns cgroup path(s) to save in a state file in order to
	// restore later.
	//
	// For cgroup v1, a key is cgroup subsystem name, and the value is the
	// path to the cgroup for this subsystem.
	//
	// For cgroup v2 unified hierarchy, a key is "", and the value is the
	// unified path.
	GetPaths() map[string]string

	// GetCgroups returns the cgroup data as configured.
	GetCgroups() (*Cgroup, error)

	// GetFreezerState retrieves the current FreezerState of the cgroup.
	GetFreezerState() (FreezerState, error)

	// Exists returns whether the cgroup path exists or not.
	Exists() bool

	// OOMKillCount reports OOM kill count for the cgroup.
	OOMKillCount() (uint64, error)
}
