package v1

import (
	"net"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/util/ip"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient

type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              models.KlusterSpec   `json:"spec"`
	Status            models.KlusterStatus `json:"status,omitempty"`
}

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
		return nil, err
	}
	ip, err := ip.GetIndexedIP(ipnet, 1)
	if err != nil {
		return nil, err
	}
	return ip, nil

}
