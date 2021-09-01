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
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("get user command in run container")
	}
	err := setUpMount()
	if err != nil {
		logrus.Errorf("set up mount, err: %v", err)
		return err
	}

	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("look %s path, err: %v", cmdArray[0], err)
		return err
	}

	err = syscall.Exec(path, cmdArray[0:], os.Environ())
	if err != nil {
		return err
	}
	return nil
}

func readUserCommand() []string {
	// cmd.ExtraFiles readPipe
	pipe := os.NewFile(uintptr(3), "pipe")
	bs, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("read pipe, err : %v", err)
		return nil
	}
	msg := string(bs)
	return strings.Split(msg, " ")
}

func setUpMount() error {
	err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return err
	}

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount proc, err: %v", err)
		return err
	}
	return nil
}
