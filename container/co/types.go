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
package co

type Spec struct {
	Root        string            `json:"root"`
	Hostname    string            `json:"hostname"`
	Annotations map[string]string `json:"annotations"`
	Process     *Process
	// Linux       *Linux
	Mounts []Mount
}

type Process struct {
	CommandLine     string   `json:"command_line"`
	Args            string   `json:"args"`
	Env             []string `json:"env"`
	Cwd             string   `json:"cwd"`
	OOMScoreAdj     *int     `json:"oom_score_adj"`
	NoNewPrivileges bool     `json:"no_new_privileges"`
	Namespace       string   `json:"namespace"`
}

// Mount specifies a mount for a container.
type Mount struct {
	// Destination is the absolute path where the mount will be placed in the container.
	Destination string `json:"destination"`
	// Type specifies the mount kind.
	Type string `json:"type,omitempty"`
	// Source specifies the source path of the mount.
	Source string `json:"source,omitempty"`
	// Options are fstab style mount options.
	Options []string `json:"options,omitempty"`
}
