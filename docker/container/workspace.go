/*
   Copyright @ 2021 bocloud <fushaosong@beyondcent.com>.

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
	"github.com/go-kinds/docker/common"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path"
	"strings"
)

func NewWorkSpace(volume, containerName, imageName string) error {
	err := createReadOnlyLayer(imageName)
	if err != nil {
		logrus.Errorf("create read only layer, err :%v", err)
		return err
	}

	err = createWriteLayer(containerName)
	if err != nil {
		logrus.Errorf("create write layer, err: %v", err)
		return err
	}

	err = CreateMountPoint(containerName, imageName)
	if err != nil {
		logrus.Errorf("create mount point, err: %v", err)
		return err
	}
	mountVolume(containerName, imagename, volume)
	return nil
}

func createReadOnlyLayer(imageName string) error {
	imagePath := path.Join(common.RootPath, imageName)
	_, err := os.Stat(imagePath)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(imagePath, os.ModePerm)
		if err != nil {
			logrus.Errorf("mkdir image path, err: %v", err)
			return err
		}
	}
	imageTarPath := path.Join(common.RootPath, fmt.Sprintf("%s.tar", imageName))
	if _, err = exec.Command("tar", "-xvf", imageTarPath, "-C", imagePath).CombinedOutput(); err != nil {
		logrus.Errorf("tar image tar,path: %s, err: %v", imageTarPath, err)
		return err
	}
	return nil
}

func createWriteLayer(containerName string) error {
	writeLayerPath := path.Join(common.RootPath, common.WriteLayer, containerName)
	_, err := os.Stat(writeLayerPath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(writeLayerPath, os.ModePerm)
		if err != nil {
			logrus.Errorf("mkdir write layer, err: %v", err)
			return err
		}
	}
	return nil
}

func CreateMountPoint(containerName, imageName string) error {
	mntPath := path.Join(common.MntPath, containerName)
	_, err := os.Stat(mntPath)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(mntPath, os.ModePerm)
		if err != nil {
			logrus.Errorf("mkdir mnt path, err: %v", err)
			return err
		}
	}

	writeLayPath := path.Join(common.RootPath, common.WriteLayer, containerName)
	imagePath := path.Join(common.RootPath, imageName)
	dirs := fmt.Sprintf("dirs=%s:%s", writeLayPath, imagePath)
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "node", mntPath)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mnt cmd run, err: %v", err)
		return err
	}
	return nil
}

func mountVolume(containerName, imageName, volume string) {
	if volume != "" {
		volumes := strings.Split(volume, ":")
		if len(volumes) > 1 {
			parentPath := volumes[0]
			if _, err := os.Stat(parentPath); err != nil && os.IsNotExist(err) {
				if err := os.MkdirAll(parentPath, os.ModePerm); err != nil {
					logrus.Errorf("mkdir parent path: %s, err: %v", parentPath, err)
				}
			}

			containerPath := volumes[1]
			containerVolumePath := path.Join(common.MntPath, containerName, containerPath)
			if _, err := os.Stat(containerVolumePath); err != nil && os.IsNotExist(err) {
				if err := os.MkdirAll(containerVolumePath, os.ModePerm); err != nil {
					logrus.Errorf("mkdir volume path path: %s, err: %v", containerVolumePath, err)
				}
			}

			dirs := fmt.Sprintf("dirs=%s", parentPath)
			cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				logrus.Errorf("mount cmd run, err: %v", err)
			}
		}
	}
}

func DeleteWorkSpace(containerName, volume string) error {
	err := unMountPoint(containerName)
	if err != nil {
		return err
	}

	err = deleteWriteLayer(containerName)
	if err != nil {
		return err
	}

	deleteVolume(containerName, volume)
	return nil
}

func unMountPoint(containerName string) error {
	mntPath := path.Join(common.MntPath, containerName)
	if _, err := exec.Command("umount", mntPath).CombinedOutput(); err != nil {
		logrus.Errorf("umount mnt, err : %v", err)
		return err
	}
	err := os.RemoveAll(mntPath)
	if err != nil {
		logrus.Errorf("remove mnt path, err: %v", err)
		return err
	}
	return nil
}

func deleteWriteLayer(containerName string) error {
	writeLayerPath := path.Join(common.RootPath, common.WriteLayer, containerName)
	return os.RemoveAll(writeLayerPath)
}

func deleteVolume(containerName, volume string) error {
	if volume != "" {
		volumes := strings.Split(volume, ":")
		if len(volumes) > 1 {
			mntPath := path.Join(common.MntPath, containerName)
			containerPath := path.Join(mntPath, volumes[1])
			if _, err := exec.Command("umount", containerPath).CombinedOutput(); err != nil {
				logrus.Errorf("umount container path, err: %v", err)
			}
		}
	}

}
