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
package network

import (
	"encoding/json"
	"github.com/go-kinds/docker/common"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: common.DefaultAllocatorPath,
}

func (ipam IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		return err
	}

	file, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}

	defer file.Close()
	bs := make([]byte, 2000)
	n, err := file.Read(bs)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs[:n], ipam.Subnets)
}

func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(ipamConfigFileDir, os.ModePerm); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	bs, _ := json.Marshal(ipam.Subnets)
	_, err = file.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	err = ipam.load()
	if err != nil {
		logrus.Errorf("dump allocation info, err: %v", err)
		return nil, err
	}
	_, subnet, err = net.ParseCIDR(subnet.String())
	if err != nil {
		return nil, err
	}
	one, size := subnet.Mask.Size()
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}
	err = ipam.dump()
	if err != nil {
		logrus.Errorf("ALLOCATE IP, DUMP IPAM INFO, ERR: %V", err)
		return nil, err
	}
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, err := net.ParseCIDR(subnet.String())
	if err != nil {
		return err
	}

	err = ipam.load()
	if err != nil {
		logrus.Errorf("dump allocation info, err: %v", err)
		return err
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)
	err = ipam.dump()
	if err != nil {
		logrus.Errorf("release ip, dump ipam info, err: %v", err)
	}
	return nil

}
