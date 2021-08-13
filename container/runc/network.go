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
package runc

type Network struct {
	// Type sets the networks type, commonly veth and loopback
	Type string `json:"type"`

	// Name of the network interface
	Name string `json:"name"`

	// The bridge to use.
	Bridge string `json:"bridge"`

	// MacAddress contains the MAC address to set on the network interface
	MacAddress string `json:"mac_address"`

	// Address contains the IPv4 and mask to set on the network interface
	Address string `json:"address"`

	// Gateway sets the gateway address that is used as the default for the interface
	Gateway string `json:"gateway"`

	// IPv6Address contains the IPv6 and mask to set on the network interface
	IPv6Address string `json:"ipv6_address"`

	// IPv6Gateway sets the ipv6 gateway address that is used as the default for the interface
	IPv6Gateway string `json:"ipv6_gateway"`

	// Mtu sets the mtu value for the interface and will be mirrored on both the host and
	// container's interfaces if a pair is created, specifically in the case of type veth
	// Note: This does not apply to loopback interfaces.
	Mtu int `json:"mtu"`

	// TxQueueLen sets the tx_queuelen value for the interface and will be mirrored on both the host and
	// container's interfaces if a pair is created, specifically in the case of type veth
	// Note: This does not apply to loopback interfaces.
	TxQueueLen int `json:"txqueuelen"`

	// HostInterfaceName is a unique name of a veth pair that resides on in the host interface of the
	// container.
	HostInterfaceName string `json:"host_interface_name"`

	// HairpinMode specifies if hairpin NAT should be enabled on the virtual interface
	// bridge port in the case of type veth
	// Note: This is unsupported on some systems.
	// Note: This does not apply to loopback interfaces.
	HairpinMode bool `json:"hairpin_mode"`
}
