/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// func main() {
// 	cmd := exec.Command("sh")
// 	cmd.SysProcAttr = &syscall.SysProcAttr{
// 		Cloneflags: syscall.CLONE_NEWUTS |
// 			syscall.CLONE_NEWIPC |
// 			syscall.CLONE_NEWPID |
// 			syscall.CLONE_NEWNS |
// 			syscall.CLONE_NEWUSER |
// 			syscall.CLONE_NEWNET,
// 		UidMappings: []syscall.SysProcIDMap{
// 			{
// 				ContainerID: 1,
// 				HostID:      0,
// 				Size:        1,
// 			},
// 		},
// 		GidMappings: []syscall.SysProcIDMap{
// 			{
// 				ContainerID: 1,
// 				HostID:      0,
// 				Size:        1,
// 			},
// 		},
// 	}
// 	cmd.Stdin = os.Stdin
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
//
// 	if err := cmd.Run(); err != nil {
// 		log.Fatal(err)
// 	}
//
// }

const (
	cgroupMemoryHierarchyMount = "/sys/fs/cgroup/memory"
)

func main() {
	fmt.Println(strings.Join(os.Args, ","))

	if os.Args[0] == "/proc/self/exe" {
		// container proc
		fmt.Printf("current pid %d \n", syscall.Getpid())
		cmd := exec.Command("sh", "-c", "stress --vm-bytes 200m --vm-keep -m 1")
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}

	cmd := exec.Command("/proc/self/exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", cmd.Process.Pid)

	newCgroup := path.Join(cgroupMemoryHierarchyMount, "cgroup-demo-memroy")
	_ = os.Remove(newCgroup)
	if err := os.Mkdir(newCgroup, 0755); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(path.Join(newCgroup, "tasks"), []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(path.Join(newCgroup, "memory.limit_in_bytes"), []byte("100m"), 0644); err != nil {
		panic(err)
	}

	cmd.Process.Wait()

}
