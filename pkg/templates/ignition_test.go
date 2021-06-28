package templates

import (
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/assert"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

var (
	testKlusterSecret kubernikusv1.Secret
	testKluster       kubernikusv1.Kluster
	imageRegistry     version.ImageRegistry
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
			ClusterCIDR:      swag.String("3.3.3.3/24"),
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

	testKlusterSecret = kubernikusv1.Secret{
		NodePassword:   "password",
		BootstrapToken: "BootstrapToken",
		Certificates: kubernikusv1.Certificates{
			TLSCACertificate:                     "TLSCACertificate",
			KubeletClientsCACertificate:          "KubeletClientsCACertificate",
			ApiserverClientsKubeProxyCertificate: "ApiserverClientsKubeProxyCertificate",
			ApiserverClientsKubeProxyPrivateKey:  "ApiserverClientsKubeProxyPrivateKey",
		},
		Openstack: kubernikusv1.Openstack{
			AuthURL:    "AuthURL",
			Username:   "Username",
			Password:   "Password",
			DomainName: "DomainName",
			Region:     "Region",
		},
	}

	imageRegistry = version.ImageRegistry{
		Versions: map[string]version.KlusterVersion{
			"1.19": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.19"}},
			"1.18": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.18"}},
			"1.17": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.17"}},
			"1.16": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.16"}},
			"1.15": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.15"}},
			"1.14": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.14"}},
			"1.13": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.13"}},
			"1.12": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.12"}},
			"1.11": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.11"}},
			"1.10": {Hyperkube: version.ImageVersion{Repository: "nase", Tag: "v1.10"}},
		},
	}

}

func TestGenerateNode(t *testing.T) {

	kluster := testKluster.DeepCopy()

	for version := range imageRegistry.Versions {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(kluster, nil, "test", &testKlusterSecret, false, imageRegistry, log.NewNopLogger())
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

	data, err := Ignition.GenerateNode(kluster, pool, "test", &testKlusterSecret, false, imageRegistry, log.NewNopLogger())
	if assert.NoError(t, err, "Failed to generate node") {
		//Ensure we rendered the expected template
		assert.Contains(t, string(data), fmt.Sprintf("--node-labels=ccloud.sap.com/nodepool=%s", pool.Name))
	}

	gpuPool := &models.NodePool{Name: "some-name", Flavor: "zghuh"}
	data, err = Ignition.GenerateNode(kluster, gpuPool, "test", &testKlusterSecret, false, imageRegistry, log.NewNopLogger())
	if assert.NoError(t, err, "Failed to generate node") {
		//Ensure we rendered the expected template
		assert.Contains(t, string(data), fmt.Sprintf("--node-labels=ccloud.sap.com/nodepool=%s,gpu=nvidia-tesla-v100", pool.Name))
		assert.Contains(t, string(data), fmt.Sprintf("--register-with-taints=nvidia.com/gpu=present:NoSchedule"))
	}
}
