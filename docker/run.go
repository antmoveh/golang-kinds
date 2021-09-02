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
package main

import (
	"github.com/go-kinds/docker/cgroups"
	"github.com/go-kinds/docker/cgroups/subsystem"
	"github.com/go-kinds/docker/container"
	"github.com/go-kinds/docker/network"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

func Run(cmdArray []string, tty bool, res *subsystem.ResourceConfig, containerName, imageName, volume, net string, envs, ports []string) {
	parent, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, envs)
	if parent == nil {
		logrus.Errorf("failed to new parent process")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Errorf("parent start failed, err: %v", err)
		return
	}
	cgroupManager := cgroups.NewCGroupManager("go-docker")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	if net != "" {
		err := network.Init()
		if err != nil {
			logrus.Errorf("network init failed, err: %v", err)
			return
		}
		containerInfo := &container.ContainerInfo{
			Id:          "containerID",
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: ports,
		}
		if err := network.Connect(net, containerInfo); err != nil {
			logrus.Errorf("connect network, err: %v", err)
			return
		}
	}

	//  write cmd to pipe when init start
	sendInitCommand(cmdArray, writePipe)
	parent.Wait()

	err := container.DeleteWorkSpace(containerName, volume)
	if err != nil {
		logrus.Errorf("delete work space, err: %v", err)
	}
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	logrus.Info("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
