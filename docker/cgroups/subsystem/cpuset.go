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

type CpuSetSubSystem struct {
	apply bool
}

func (*CpuSetSubSystem) Name() string {
	return "cpuset"
}

func (c *CpuSetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		logrus.Errorf("get %s path, err: %v", cgroupPath, err)
		return err
	}
	if res.CpuSet != "" {
		c.apply = true
		err := ioutil.WriteFile(path.Join(subsystemCgroupPath, "cpuset.cpus"), []byte(res.CpuSet), 0644)
		if err != nil {
			logrus.Errorf("failed to write file cpuset.cpus, err: %+v", err)
			return err
		}
	}
	return nil
}

func (c *CpuSetSubSystem) Remove(cgroupPath string) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	return os.RemoveAll(subsystemCgroupPath)
}

func (c *CpuSetSubSystem) Apply(cgroupPath string, pid int) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	taskPath := path.Join(subsystemCgroupPath, "tasks")
	err = ioutil.WriteFile(taskPath, []byte(strconv.Itoa(pid)), os.ModePerm)
	if err != nil {
		logrus.Errorf("write pid to tasks, path: %s, pid: %d, err: %v", taskPath, pid, err)
		return err
	}
	return nil
}
