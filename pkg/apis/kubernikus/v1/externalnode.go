package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ExternalNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ExternalNodeSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ExternalNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ExternalNode `json:"items"`
}

type ExternalNodeSpec struct {
	IPXE     string               `json:"ipxe"`
	Networks []SystemdNetworkSpec `json:"networks,omitempty"`
	Netdevs  []SystemdNetDevSpec  `json:"netdevs,omitempty"`
}

type SystemdNetworkSpec struct {
	Name    string                 `json:"name"`
	Match   *SystemdNetworkMatch   `json:"match,omitempty"`
	Network *SystemdNetworkNetwork `json:"network,omitempty"`
}

type SystemdNetworkMatch struct {
	Name       string `json:"name,omitempty"`
	MACAddress string `json:"macAddress,omitempty"`
}

type SystemdNetworkNetwork struct {
	DHCP    string   `json:"dhcp,omitempty"`
	Address string   `json:"address,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,inline,omitempty"`
	Domains string   `json:"domains,omitempty"`
	LLDP    string   `json:"lldp,omitempty"`
	Bond    string   `json:"bond,omitempty"`
}

type SystemdNetDevSpec struct {
	Name   string               `json:"name"`
	NetDev *SystemdNetDevNetDev `json:"netdev,omitempty"`
	Bond   *SystemdNetDevBond   `json:"bond,omitempty"`
}

type SystemdNetDevNetDev struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	MTUBytes int    `json:"mtuBytes,omitempty"`
}

type SystemdNetDevBond struct {
	Mode             string `json:"mode,omitempty"`
	MIMMonitorSec    string `json:"mimMonitorSec,omitempty"`
	LACPTransmitRate string `json:"lacpTransmitRate,omitempty"`
	UpDelaySec       string `json:"upDelaySec,omitempty"`
	DownDelaySec     string `json:"downDelaySec,omitempty"`
	MinLinks         int    `json:"minLinks,omitempty"`
}
