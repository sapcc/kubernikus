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

var (
	testKlusterSecret v1.Secret
	testKluster       kubernikusv1.Kluster
)

func init() {
	//speed up tests by lowering hash rounds during testing
	passwordHashRounds = sha512_crypt.RoundsMin

	//test data
	testKluster = kubernikusv1.Kluster{
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

	testKlusterSecret = v1.Secret{
		ObjectMeta: testKluster.ObjectMeta,
		Data:       secretData,
	}

}

func TestGenerateNode(t *testing.T) {

	kluster := testKluster.DeepCopy()

	for _, version := range []string{"1.7", "1.8", "1.9", "1.10"} {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(kluster, nil, "test", &testKlusterSecret, log.NewNopLogger())
		if assert.NoError(t, err, "Failed to generate node for version %s", version) {
			//Ensure we rendered the expected template
			assert.Contains(t, string(data), fmt.Sprintf("KUBELET_IMAGE_TAG=v%s", version))
		}
	}
}

func TestNodeLabels(t *testing.T) {
	kluster := testKluster.DeepCopy()
	kluster.Spec.Version = "1.10"

	pool := &models.NodePool{Name: "some-name"}

	data, err := Ignition.GenerateNode(kluster, pool, "test", &testKlusterSecret, log.NewNopLogger())
	if assert.NoError(t, err, "Failed to generate node") {
		//Ensure we rendered the expected template
		assert.Contains(t, string(data), fmt.Sprintf("--node-labels=ccloud.sap.com/nodepool=%s", pool.Name))
	}

	gpuPool := &models.NodePool{Name: "some-name", Flavor: "zghuh"}
	data, err = Ignition.GenerateNode(kluster, gpuPool, "test", &testKlusterSecret, log.NewNopLogger())
	if assert.NoError(t, err, "Failed to generate node") {
		//Ensure we rendered the expected template
		assert.Contains(t, string(data), fmt.Sprintf("--node-labels=ccloud.sap.com/nodepool=%s,gpu=nvidia-tesla-v100", pool.Name))
	}
}
