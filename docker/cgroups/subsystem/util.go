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
	"bufio"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRootPath, err := findCgroupMountPoint(subsystem)
	if err != nil {
		logrus.Errorf("find cgroup mount point, err :%s", err.Error())
		return "", err
	}
	cgroupTotalPath := path.Join(cgroupRootPath, cgroupPath)
	_, err = os.Stat(cgroupTotalPath)
	if err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(cgroupTotalPath, 0755); err != nil {
			return "", err
		}
	}
	return cgroupTotalPath, nil
}

func findCgroupMountPoint(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem && len(fields) > 4 {
				return fields[4], nil
			}
		}
	}
	return "", scanner.Err()
}
