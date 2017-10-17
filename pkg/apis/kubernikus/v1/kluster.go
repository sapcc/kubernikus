package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodePoolConfig struct {
	Upgrade bool `json:"upgrade"`
	Repair  bool `json:"repair"`
}

type NodePool struct {
	Name   string         `json:"name"`
	Size   int            `json:"size"`
	Flavor string         `json:"flavor"`
	Image  string         `json:"image"`
	Config NodePoolConfig `json:"config"`
}

type OpenstackSpec struct {
	ProjectID  string `json:"projectID"`
	RouterID   string `json:"routerID"`
	NetworkID  string `json:"networkID"`
	LBSubnetID string `json:"lbSubnetID"`
}

type KlusterSpec struct {
	Name             string        `json:"name"`
	Domain           string        `json:"domain"`
	ClusterCIDR      string        `json:"clusterCIDR"`
	ClusterDNS       string        `json:"clusterDNS"`
	ClusterDNSDomain string        `json:"clusterDNSDomain"`
	ServiceCIDR      string        `json:"serviceCIDR"`
	AdvertiseAddress string        `json:"advertiseAddress"`
	NodePools        []NodePool    `json:"nodePools,omitempty"`
	Openstack        OpenstackSpec `json:"openstack,omitempty"`
}

type KlusterState string

const (
	KlusterPending     KlusterState = "Pending"
	KlusterCreating    KlusterState = "Creating"
	KlusterReady       KlusterState = "Ready"
	KlusterTerminating KlusterState = "Terminating"
	KlusterTerminated  KlusterState = "Terminated"
	KlusterError       KlusterState = "Error"
)

type KlusterStatus struct {
	Kluster   KlusterInfo    `json:"kluster"`
	Apiserver string         `json:"apiserver"`
	Wormhole  string         `json:"wormhole"`
	NodePools []NodePoolInfo `json:"nodePools,omitempty"`
}

type KlusterInfo struct {
	State   KlusterState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type NodePoolInfo struct {
	Name        string `json:"name"`
	Size        int    `json:size`
	Running     int    `json:running`
	Healthy     int    `json:healthy`
	Schedulable int    `json:schedulable`
}

// +genclient

type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              KlusterSpec   `json:"spec"`
	Status            KlusterStatus `json:"status,omitempty"`
}

type KlusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Kluster `json:"items"`
}

func (spec KlusterSpec) Validate() error {
	//Add some validation
	return nil
}

func (spec Kluster) Account() string {
	return spec.ObjectMeta.Labels["account"]
}
