package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

type KlusterSpec struct {
	Name      string     `json:"name"`
	NodePools []NodePool `json:"nodePools"`
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
	State   KlusterState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
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
