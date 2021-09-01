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
package container

import (
	"github.com/go-kinds/docker/common"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(tty bool, volume, containerName, imageName string, envs []string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, _ := os.Pipe()
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Env = append(os.Environ(), envs...)
	cmd.ExtraFiles = []*os.File{readPipe}

	err := NewWorkSpace(volume, containerName, imageName)
	if err != nil {
		logrus.Errorf("new work space, err : %v", err)
	}
	cmd.Dir = common.MntPath
	return cmd, writePipe
}
