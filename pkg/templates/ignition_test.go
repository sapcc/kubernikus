package templates

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

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
				LBSubnetID: "lb-id",
				RouterID:   "router-id",
			},
		},
		Status: models.KlusterStatus{
			Apiserver: "https://apiserver.somewhere",
		},
	}
	secretData := make(map[string][]byte, len(Ignition.requiredNodeSecrets))
	for _, f := range Ignition.requiredNodeSecrets {
		secretData[f] = []byte(fmt.Sprintf("[DATA for %s]", f))
	}

	secret := v1.Secret{
		ObjectMeta: kluster.ObjectMeta,
		Data:       secretData,
	}
	bytes, err := Ignition.GenerateNode(&kluster, &secret)
	assert.NoError(t, err)

	fmt.Printf("bytes = %+v\n", string(bytes))

}
