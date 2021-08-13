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

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func startContainer() {

}

func prepareContainerSpec() {

	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(file)

	spce := &Spec{
		Root:     filepath.Join(file, "rootfs"),
		Hostname: "dev",
		Annotations: map[string]string{
			"version": "1",
		},
		Process: &Process{
			CommandLine:     "bash",
			Args:            "",
			Env:             nil,
			Cwd:             "/",
			OOMScoreAdj:     nil,
			NoNewPrivileges: false,
		},
		Mounts: nil,
	}
}
