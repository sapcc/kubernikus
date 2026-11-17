package templates

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/go-openapi/swag/conv"
	"github.com/stretchr/testify/assert"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
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
			ClusterCIDR:      conv.Pointer("3.3.3.3/24"),
			ServiceCIDR:      "4.4.4.4/24",
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
		Openstack: kubernikusv1.Openstack{
			AuthURL:    "AuthURL",
			Username:   "Username",
			Password:   "Password",
			DomainName: "DomainName",
			Region:     "Region",
		},
	}
	_, err := util.NewCertificateFactory(&testKluster, &testKlusterSecret.Certificates, "kubernikus.test").Ensure()
	if err != nil {
		panic(err)
	}

	imageRegistry = version.ImageRegistry{
		Versions: map[string]version.KlusterVersion{
			"1.30": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.30"}},
			"1.29": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.29"}},
			"1.28": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.28"}},
			"1.27": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.27"}},
			"1.26": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.26"}},
			"1.24": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.24"}},
			"1.21": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.21"}},
			"1.20": {Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.20"}},
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

	kluster.Spec.SSHPublicKey = strings.Repeat("a", 10000) //max out ssh key

	for version := range imageRegistry.Versions {
		kluster.Spec.Version = version
		data, err := Ignition.GenerateNode(kluster, nil, "test", "abc123", &testKlusterSecret, false, imageRegistry, log.NewNopLogger())
		if assert.NoError(t, err, "Failed to generate node for version %s", version) {
			//Ensure we rendered the expected template
			assert.Contains(t, string(data), fmt.Sprintf("v%s", version))
			userData := base64.StdEncoding.EncodeToString(data)
			assert.LessOrEqualf(t, len(userData), 65535, "userdata exceeds openstack limit for api version %s template", version)
		}
	}
}

func TestNodeLabels(t *testing.T) {
	kluster := testKluster.DeepCopy()
	kluster.Spec.Version = "1.30"

	pool := &models.NodePool{Name: "some-name"}

	data, err := Ignition.GenerateNode(kluster, pool, "test", "abc123", &testKlusterSecret, false, imageRegistry, log.NewNopLogger())
	if assert.NoError(t, err, "Failed to generate node") {
		//Ensure we rendered the expected template
		assert.Contains(t, string(data), fmt.Sprintf("--node-labels=kubernikus.cloud.sap/cni=true,ccloud.sap.com/nodepool=%s", pool.Name))
	}
}

func TestServerTLSBootstrap(t *testing.T) {
	registry := version.ImageRegistry{Versions: map[string]version.KlusterVersion{}}
	for k, v := range imageRegistry.Versions {
		registry.Versions[k] = v
	}
	registry.Versions["1.35"] = version.KlusterVersion{Kubelet: version.ImageVersion{Repository: "nase", Tag: "v1.35"}}

	cases := []struct {
		annotations map[string]string
		expected    bool
	}{
		{nil, false},
		{map[string]string{kubernikusv1.KubeletServingCertificatesAnnotationKey: "false"}, false},
		{map[string]string{kubernikusv1.KubeletServingCertificatesAnnotationKey: "true"}, true},
		{map[string]string{kubernikusv1.KubeletServingCertificatesAnnotationKey: " TRUE "}, true},
	}

	for _, kubeVersion := range []string{"1.27", "1.30", "1.35"} {
		for _, c := range cases {
			kluster := testKluster.DeepCopy()
			kluster.Spec.Version = kubeVersion
			kluster.Annotations = c.annotations

			data, err := Ignition.GenerateNode(kluster, nil, "test", "abc123", &testKlusterSecret, false, registry, log.NewNopLogger())
			if !assert.NoError(t, err, "Failed to generate node for version %s", kubeVersion) {
				continue
			}
			kubeletConfig := ignitionFileContents(t, data, "/etc/kubernetes/kubelet/config")
			if c.expected {
				assert.Contains(t, kubeletConfig, "rotateCertificates: true\nserverTLSBootstrap: true\n", "version %s annotations %v", kubeVersion, c.annotations)
			} else {
				assert.Contains(t, kubeletConfig, "rotateCertificates: true\nnodeLeaseDurationSeconds: 20", "version %s", kubeVersion)
				assert.NotContains(t, kubeletConfig, "serverTLSBootstrap", "version %s annotations %v", kubeVersion, c.annotations)
			}
		}
	}
}

// ignitionFileContents returns the decoded contents of a file from rendered ignition JSON
func ignitionFileContents(t *testing.T, data []byte, path string) string {
	var config struct {
		Storage struct {
			Files []struct {
				Path     string `json:"path"`
				Contents struct {
					Compression string `json:"compression"`
					Source      string `json:"source"`
				} `json:"contents"`
			} `json:"files"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse ignition: %s", err)
	}
	for _, f := range config.Storage.Files {
		if f.Path != path {
			continue
		}
		header, payload, found := strings.Cut(strings.TrimPrefix(f.Contents.Source, "data:"), ",")
		if !found {
			t.Fatalf("unexpected source for %s: %s", path, f.Contents.Source)
		}
		var raw []byte
		var err error
		if strings.HasSuffix(header, ";base64") {
			raw, err = base64.StdEncoding.DecodeString(payload)
		} else {
			var s string
			s, err = url.PathUnescape(payload)
			raw = []byte(s)
		}
		if err != nil {
			t.Fatalf("failed to decode %s: %s", path, err)
		}
		if f.Contents.Compression == "gzip" {
			r, err := gzip.NewReader(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("failed to gunzip %s: %s", path, err)
			}
			if raw, err = io.ReadAll(r); err != nil {
				t.Fatalf("failed to gunzip %s: %s", path, err)
			}
		}
		return string(raw)
	}
	t.Fatalf("file %s not found in ignition", path)
	return ""
}
