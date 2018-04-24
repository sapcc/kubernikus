package templates

import (
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func init() {
	//speed up tests by lowering hash rounds during testing
	passwordHashRounds = sha512_crypt.RoundsMin
}

func TestGenerateNode(t *testing.T) {

	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: models.KlusterSpec{
			SSHPublicKey:     "ssh-rsa nasenbaer bla@fasel",
			AdvertiseAddress: "1.1.1.1",
			ClusterCIDR:      "3.3.3.3/24",
			DNSAddress:       "2.2.2.2",
			DNSDomain:        "cluster.local",
			Openstack: models.OpenstackSpec{
				LBSubnetID:          "lb-id",
				LBFloatingNetworkID: "lb-fipid",
				RouterID:            "router-id",
			},
		},
		Status: models.KlusterStatus{
			Apiserver: "https://apiserver.somewhere",
		},
	}
	secretData := make(map[string][]byte, len(Ignition.requiredNodeSecrets)+1)
	for _, f := range Ignition.requiredNodeSecrets {
		secretData[f] = []byte(fmt.Sprintf("[DATA for %s]", f))
	}
	secretData["node-password"] = []byte("password")

	secret := v1.Secret{
		ObjectMeta: kluster.ObjectMeta,
		Data:       secretData,
	}

	for _, version := range []string{"1.7", "1.8", "1.9", "1.10"} {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(&kluster, "test", &secret, nil, log.NewNopLogger())
		if assert.NoError(t, err, "Failed to generate node for version %s", version) {
			//Ensure we rendered the expected template
			assert.Contains(t, string(data), fmt.Sprintf("KUBELET_IMAGE_TAG=v%s", version))
		}
		//fmt.Printf("data = %+v\n", string(data))
	}
}

func TestGenerateNodeBareMetal(t *testing.T) {

	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: models.KlusterSpec{
			SSHPublicKey:     "ssh-rsa nasenbaer bla@fasel",
			AdvertiseAddress: "1.1.1.1",
			ClusterCIDR:      "3.3.3.3/24",
			DNSAddress:       "2.2.2.2",
			DNSDomain:        "cluster.local",
			Openstack: models.OpenstackSpec{
				LBSubnetID:          "lb-id",
				LBFloatingNetworkID: "lb-fipid",
				RouterID:            "router-id",
			},
		},
		Status: models.KlusterStatus{
			Apiserver: "https://apiserver.somewhere",
		},
	}

	externalNode := &kubernikusv1.ExternalNode{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExternalNode",
			APIVersion: "kubernikus.sap.cc/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "master0",
			Namespace: "test",
		},
		Spec: kubernikusv1.ExternalNodeSpec{
			IPXE: "12:3A:7D:6A:48:F1",
			Networks: []kubernikusv1.SystemdNetworkSpec{
				kubernikusv1.SystemdNetworkSpec{
					Match: &kubernikusv1.SystemdNetworkMatch{
						Name:       "eth0",
						MACAddress: "12:df:2f:67:1e:85",
					},
					Network: &kubernikusv1.SystemdNetworkNetwork{
						DHCP: "no",
						LLDP: "yes",
						Bond: "bond1",
					},
				},
				kubernikusv1.SystemdNetworkSpec{
					Match: &kubernikusv1.SystemdNetworkMatch{
						Name:       "eth1",
						MACAddress: "12:df:2f:67:1e:86",
					},
					Network: &kubernikusv1.SystemdNetworkNetwork{
						DHCP: "no",
						LLDP: "yes",
						Bond: "bond1",
					},
				},
				kubernikusv1.SystemdNetworkSpec{
					Match: &kubernikusv1.SystemdNetworkMatch{
						Name: "bond1",
					},
					Network: &kubernikusv1.SystemdNetworkNetwork{
						DHCP:    "no",
						Address: "1.4.7.2/29",
						Gateway: "1.4.7.1",
						DNS:     []string{"1.2.9.200", "1.2.9.201"},
						Domains: "bla.cloud.sap",
					},
				},
			},
			Netdevs: []kubernikusv1.SystemdNetDevSpec{
				kubernikusv1.SystemdNetDevSpec{
					Name: "bond1",
					NetDev: &kubernikusv1.SystemdNetDevNetDev{
						Name:     "bond1",
						Kind:     "bond",
						MTUBytes: 9000,
					},
					Bond: &kubernikusv1.SystemdNetDevBond{
						Mode:             "802.3ad",
						MIMMonitorSec:    "1s",
						LACPTransmitRate: "fast",
						UpDelaySec:       "3s",
						DownDelaySec:     "3s",
						MinLinks:         1,
					},
				},
			},
		},
	}

	secretData := make(map[string][]byte, len(Ignition.requiredNodeSecrets)+1)
	for _, f := range Ignition.requiredNodeSecrets {
		secretData[f] = []byte(fmt.Sprintf("[DATA for %s]", f))
	}
	secretData["node-password"] = []byte("password")

	secret := v1.Secret{
		ObjectMeta: kluster.ObjectMeta,
		Data:       secretData,
	}

	for _, version := range []string{"1.9"} {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(&kluster, "test", &secret, externalNode, log.NewNopLogger())
		if assert.NoError(t, err, "Failed to generate node for version %s", version) {
			//Ensure we rendered the expected template
			assert.Contains(t, string(data), fmt.Sprintf("KUBELET_IMAGE_TAG=v%s", version))
		}
		//fmt.Printf("data = %+v\n", string(data))
	}
}
