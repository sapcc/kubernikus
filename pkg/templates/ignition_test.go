package templates

import (
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func TestGenerateNode(t *testing.T) {

	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: models.KlusterSpec{
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

	for _, version := range []string{"1.7", "1.8", "1.9"} {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(&kluster, &secret, log.NewNopLogger())
		if assert.NoError(t, err, "Failed to generate node for version %s", version) {
			//Ensure we rendered the expected template
			assert.Contains(t, string(data), fmt.Sprintf("KUBELET_IMAGE_TAG=v%s", version))
		}
		//fmt.Printf("data = %+v\n", string(data))
	}
}
