package v1

import (
	"fmt"
	"net"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/util/ip"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              models.KlusterSpec   `json:"spec"`
	Status            models.KlusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KlusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Kluster `json:"items"`
}

func (spec Kluster) Account() string {
	return spec.ObjectMeta.Labels["account"]
}

func (spec Kluster) ApiServiceIP() (net.IP, error) {
	_, ipnet, err := net.ParseCIDR(spec.Spec.ServiceCIDR)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse service CIDR: %s", err)
	}
	ip, err := ip.GetIndexedIP(ipnet, 1)
	if err != nil {
		return nil, err
	}
	return ip, nil

}

func (k *Kluster) NeedsFinalizer(finalizer string) bool {
	if k.ObjectMeta.DeletionTimestamp != nil {
		// already deleted. do not add another finalizer anymore
		return false
	}

	for _, f := range k.ObjectMeta.Finalizers {
		if f == finalizer {
			// Finalizer is already present, nothing to do
			return false
		}
	}

	return true
}

func (k *Kluster) HasFinalizer(finalizer string) bool {
	if k.ObjectMeta.DeletionTimestamp == nil {
		// not deleted. do not remove finalizers at this time
		return false
	}

	for _, f := range k.ObjectMeta.Finalizers {
		if f == finalizer {
			// Finalizer is already present
			return true
		}
	}

	return false
}

func (k *Kluster) AddFinalizer(finalizer string) {
	if k.NeedsFinalizer(finalizer) {
		k.Finalizers = append(k.Finalizers, finalizer)
	}
}

func (k *Kluster) RemoveFinalizer(finalizer string) {
	if k.HasFinalizer(finalizer) {
		for i, f := range k.Finalizers {
			if f == finalizer {
				k.Finalizers = append(k.Finalizers[:i], k.Finalizers[i+1:]...)
				break
			}
		}
	}
}

func (k *Kluster) Disabled() bool {
	return k.Status.MigrationsPending
}
