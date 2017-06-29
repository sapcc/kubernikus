package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const KlusterResourcePlural = "klusters"

type KlusterSpec struct {
	Name    string `json:"name"`
	Account string `json:"account"`
}

func (spec KlusterSpec) Validate() error {
	//Add some validation
	return nil
}

type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              KlusterSpec   `json:"spec"`
	Status            KlusterStatus `json:"status,omitempty"`
}

type KlusterState string

const (
	KlusterPending KlusterState = "Pending"
	KlusterCreated KlusterState = "Created"
	klusterError   KlusterState = "Error"
)

type KlusterStatus struct {
	State   KlusterState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type KlusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Kluster `json:"items"`
}
