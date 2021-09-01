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
package cgroups

import (
	"github.com/go-kinds/docker/cgroups/subsystem"
	"github.com/sirupsen/logrus"
)

type CGroupManager struct {
	Path string
}

func NewCGroupManager(path string) *CGroupManager {
	return &CGroupManager{Path: path}
}

func (c *CGroupManager) Set(res *subsystem.ResourceConfig) {
	for _, subsystem := range subsystem.Subsystems {
		err := subsystem.Set(c.Path, res)
		if err != nil {
			logrus.Errorf("set %s err: %v", subsystem.Name(), err)
		}
	}
}

func (c *CGroupManager) Apply(pid int) {
	for _, subsystem := range subsystem.Subsystems {
		err := subsystem.Apply(c.Path, pid)
		if err != nil {
			logrus.Errorf("apply task, err: %v", err)
		}
	}
}

func (c *CGroupManager) Destroy() {
	for _, subsystem := range subsystem.Subsystems {
		err := subsystem.Remove(c.Path)
		if err != nil {
			logrus.Errorf("remove %s err :%v", subsystem.Name(), err)
		}
	}

}
