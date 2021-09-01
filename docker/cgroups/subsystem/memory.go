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
package subsystem

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type MemorySubSystem struct {
}

func (*MemorySubSystem) Name() string {
	return "memory"
}

func (m *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, true)
	if err != nil {
		logrus.Errorf("get %s path, err: %v", cgroupPath, err)
		return err
	}
	if res.MemoryLimit != "" {
		err := ioutil.WriteFile(path.Join(subsystemCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MemorySubSystem) Remove(cgroupPath string) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	return os.RemoveAll(subsystemCgroupPath)
}

func (m *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	tasksPath := path.Join(subsystemCgroupPath, "tasks")
	err = ioutil.WriteFile(tasksPath, []byte(strconv.Itoa(pid)), os.ModePerm)
	if err != nil {
		logrus.Errorf("write pid to tasks, path %s, pid: %d, err: %v", tasksPath, pid, err)
		return err
	}
	return nil
}
