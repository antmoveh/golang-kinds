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

import "os"

type PressureLevel uint

// Container is a libcontainer container object.
//
// Each container is thread-safe within the same process. Since a container can
// be destroyed by a separate process, any function may return that the container
// was not found.
type Container interface {
	BaseContainer

	// Methods below here are platform specific

	// Checkpoint checkpoints the running container's state to disk using the criu(8) utility.
	Checkpoint(criuOpts *CriuOpts) error

	// Restore restores the checkpointed container to a running state using the criu(8) utility.
	Restore(process *libProcess, criuOpts *CriuOpts) error

	// If the Container state is RUNNING or CREATED, sets the Container state to PAUSING and pauses
	// the execution of any user processes. Asynchronously, when the container finished being paused the
	// state is changed to PAUSED.
	// If the Container state is PAUSED, do nothing.
	Pause() error

	// If the Container state is PAUSED, resumes the execution of any user processes in the
	// Container before setting the Container state to RUNNING.
	// If the Container state is RUNNING, do nothing.
	Resume() error

	// NotifyOOM returns a read-only channel signaling when the container receives an OOM notification.
	NotifyOOM() (<-chan struct{}, error)

	// NotifyMemoryPressure returns a read-only channel signaling when the container reaches a given pressure level
	NotifyMemoryPressure(level PressureLevel) (<-chan struct{}, error)
}

// Status is the status of a container.
type Status int

const (
	// Created is the status that denotes the container exists but has not been run yet.
	Created Status = iota
	// Running is the status that denotes the container exists and is running.
	Running
	// Pausing is the status that denotes the container exists, it is in the process of being paused.
	Pausing
	// Paused is the status that denotes the container exists, but all its processes are paused.
	Paused
	// Stopped is the status that denotes the container does not have a created or running process.
	Stopped
)

func (s Status) String() string {
	switch s {
	case Created:
		return "created"
	case Running:
		return "running"
	case Pausing:
		return "pausing"
	case Paused:
		return "paused"
	case Stopped:
		return "stopped"
	default:
		return "unknown"
	}
}

type Factory interface {
	// Creates a new container with the given id and starts the initial process inside it.
	// id must be a string containing only letters, digits and underscores and must contain
	// between 1 and 1024 characters, inclusive.
	//
	// The id must not already be in use by an existing container. Containers created using
	// a factory with the same path (and filesystem) must have distinct ids.
	//
	// Returns the new container with a running process.
	//
	// On error, any partially created container parts are cleaned up (the operation is atomic).
	Create(id string, config *Config) (Container, error)

	// Load takes an ID for an existing container and returns the container information
	// from the state.  This presents a read only view of the container.
	Load(id string) (Container, error)

	// StartInitialization is an internal API to libcontainer used during the reexec of the
	// container.
	StartInitialization() error

	// Type returns info string about factory type (e.g. lxc, libcontainer...)
	Type() string
}

type BaseContainer interface {
	// Returns the ID of the container
	ID() string

	// Returns the current status of the container.
	Status() (Status, error)

	// State returns the current container's state information.
	// State() (*State, error)

	// OCIState returns the current container's state information.
	// OCIState() (*State, error)

	// Returns the current config of the container.
	Config() Config

	// Returns the PIDs inside this container. The PIDs are in the namespace of the calling process.
	//
	// Some of the returned PIDs may no longer refer to processes in the Container, unless
	// the Container state is PAUSED in which case every PID in the slice is valid.
	Processes() ([]int, error)

	// Returns statistics for the container.
	// Stats() (*Stats, error)

	// Set resources of container as configured
	//
	// We can use this to change resources when containers are running.
	//
	Set(config Config) error

	// Start a process inside the container. Returns error if process fails to
	// start. You can track process lifecycle with passed Process structure.
	Start(process *libProcess) (err error)

	// Run immediately starts the process inside the container.  Returns error if process
	// fails to start.  It does not block waiting for the exec fifo  after start returns but
	// opens the fifo after start returns.
	Run(process *libProcess) (err error)

	// Destroys the container, if its in a valid state, after killing any
	// remaining running processes.
	//
	// Any event registrations are removed before the container is destroyed.
	// No error is returned if the container is already destroyed.
	//
	// Running containers must first be stopped using Signal(..).
	// Paused containers must first be resumed using Resume(..).
	Destroy() error

	// Signal sends the provided signal code to the container's initial process.
	//
	// If all is specified the signal is sent to all processes in the container
	// including the initial process.
	Signal(s os.Signal, all bool) error

	// Exec signals the container to exec the users process at the end of the init.
	Exec() error
}
