package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type NodePoolConfig struct {
	Upgrade bool
	Repair  bool
}

type NodePool struct {
	Name   string
	Size   int
	Flavor string
	Image  string
	Config NodePoolConfig
}

type OpenstackInfo struct {
	ProjectID string
	RouterID  string
	NetworkID string
}

type KubernetesInfo struct {
	Server string
}

type KlusterSpec struct {
	Name           string
	OpenstackInfo  OpenstackInfo
	KubernetesInfo KubernetesInfo
	NodePools      []NodePool
}

type KlusterState string

const (
	KlusterPending     KlusterState = "Pending"
	KlusterCreating    KlusterState = "Creating"
	KlusterCreated     KlusterState = "Created"
	KlusterTerminating KlusterState = "Terminating"
	KlusterTerminated  KlusterState = "Terminated"
	KlusterError       KlusterState = "Error"
)

type KlusterStatus struct {
	State   KlusterState
	Message string
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
