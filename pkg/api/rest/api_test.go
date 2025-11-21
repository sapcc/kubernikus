package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kitlog "github.com/go-kit/log"
	errors "github.com/go-openapi/errors"
	"github.com/go-openapi/swag/conv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
	"github.com/sapcc/kubernikus/pkg/util"
)

const (
	NAMESPACE = "test"
	TOKEN     = "abc123"
	ACCOUNT   = "testaccount"
)

func init() {
	auth.DefaultPolicyFile = "../../../etc/policy.json"
}

func mockAuth(token string) (*models.Principal, error) {
	if token != TOKEN {
		return nil, errors.New(401, "auth failed")
	}
	return &models.Principal{
		ID:      "test",
		Name:    "Test Mc Dougle",
		Domain:  "TestDomain",
		Account: ACCOUNT,
		Roles:   []string{"member", "kubernetes_admin"},
	}, nil
}

func createTestHandler(t *testing.T, klusters ...runtime.Object) (http.Handler, *apipkg.Runtime, func()) {
	swaggerSpec, err := spec.Spec()
	if err != nil {
		t.Fatal(err)
	}
	api := operations.NewKubernikusAPI(swaggerSpec)
	rt := apipkg.NewRuntime(
		NAMESPACE,
		kubernikusfake.NewSimpleClientset(klusters...),
		fake.NewSimpleClientset(),
		kitlog.NewNopLogger(),
	)
	rt.KlusterClientFactory = &kubernetes.MockSharedClientFactory{
		Clientset: fake.NewSimpleClientset(),
	}
	if err := Configure(api, rt); err != nil {
		t.Fatal(err)
	}
	api.KeystoneAuth = mockAuth
	return configureAPI(api), rt, runInformer(rt)
}

func runInformer(rt *apipkg.Runtime) func() {
	closeCh := make(chan struct{})
	go rt.Informer.Run(closeCh)
	cache.WaitForCacheSync(nil, rt.Informer.HasSynced)
	return func() {
		close(closeCh)
	}
}

func createRequest(method, path, body string) *http.Request {
	var buf io.Reader
	if body != "" {
		buf = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", TOKEN)
	return req
}

func result(handler http.Handler, req *http.Request) (int, http.Header, []byte) {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	response := rec.Result()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header, body
}

func TestCreateCluster(t *testing.T) {
	handler, rt, cancel := createTestHandler(t)
	defer cancel()
	req := createRequest("POST", "/api/v1/clusters", `{"name": "nase", "spec": { "openstack": { "routerID":"routerA"}}}`)
	code, _, body := result(handler, req)
	if !assert.Equal(t, 201, code) {
		return
	}

	//Test create
	crd, err := rt.Kubernikus.KubernikusV1().Klusters(rt.Namespace).Get(context.Background(), fmt.Sprintf("%s-%s", "nase", ACCOUNT), metav1.GetOptions{})
	assert.NoError(t, err, "resource not persisted")
	assert.Equal(t, crd.Labels["account"], ACCOUNT)

	var kluster models.Kluster
	assert.NoError(t, kluster.UnmarshalBinary(body), "Failed to parse response")
	assert.Equal(t, "nase", kluster.Name)
	assert.Equal(t, "nase", kluster.Spec.Name)
	assert.Equal(t, models.KlusterPhasePending, kluster.Status.Phase)
	assert.Equal(t, kubernikus.DEFAULT_CLUSTER_CIDR, *kluster.Spec.ClusterCIDR)

	//Ensure authentication is required
	req = createRequest("POST", "/api/v1/clusters", `{"name": "nase2"}`)
	req.Header.Del("X-Auth-Token")
	code, _, _ = result(handler, req)
	assert.Equal(t, 401, code)

	//Ensure cluster CIDR does not overlap with other clusters specifying no router
	req = createRequest("POST", "/api/v1/clusters", `{"name": "ohr" }`)
	code, _, body = result(handler, req)
	if assert.Equal(t, 409, code, "response body: %s", string(body)) {
		assert.Contains(t, string(body), "nase", "when specifying no router it should always conflict with exiting clusters")
	}
	//Ensure cluster CIDR does not overlap with other clusters using the same router
	req = createRequest("POST", "/api/v1/clusters", `{"name": "ohr", "spec": { "openstack": { "routerID":"routerA"}}}`)
	code, _, body = result(handler, req)
	if assert.Equal(t, 409, code, "response body: %s", string(body)) {
		assert.Contains(t, string(body), "nase")
	}

	//Ensure specifying a different router doesn't overlap
	req = createRequest("POST", "/api/v1/clusters", `{"name": "ohr", "spec": { "openstack": { "routerID":"routerB"}}}`)
	code, _, body = result(handler, req)
	assert.Equal(t, 201, code, "specifying a different router should not conflict. response: %s", string(body))

	//Ensure we refuse service CIDRs that overlap with the control plane
	rt.Kubernikus = kubernikusfake.NewSimpleClientset()
	rt.Kubernetes = fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "198.18.128.1",
		},
	})
	req = createRequest("POST", "/api/v1/clusters", `{"name": "nase"}`)
	code, _, body = result(handler, req)
	if assert.Equal(t, 409, code) {
		assert.Contains(t, string(body), "CIDR")
	}

	//Ensure specifying an empty clusterCIDR does not fail
	req = createRequest("POST", "/api/v1/clusters", `{"name": "nocidr", "spec": { "clusterCIDR": "", "noCloud": true, "serviceCIDR":"5.5.5.5/24"}}`)

	code, _, body = result(handler, req)
	assert.Equal(t, 201, code, "Creating a cluster with empty clusterCIDR should not fail. response: %s", string(body))
	k, err := rt.Kubernikus.KubernikusV1().Klusters(NAMESPACE).Get(context.Background(), "nocidr-"+ACCOUNT, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Empty(t, k.Spec.ClusterCIDR)

}

func TestAuthenticationConfigurationValidation(t *testing.T) {
	handler, _, cancel := createTestHandler(t)
	defer cancel()
	//Ensure invalid authentication configuration is rejected
	invalidAuthConfig, err := json.Marshal(models.Kluster{
		Name: "auth",
		Spec: models.KlusterSpec{
			AuthenticationConfiguration: models.AuthenticationConfiguration(`
apiVersion: apiserver.config.k8s.io/v1beta1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: https://issuer1.example.com
    audiences:
    - audience1
    - audience2
    audienceMatchPolicy: MatchAny
`),
		},
	})
	assert.NoError(t, err, "failed to marshal authentication configuration")
	req := createRequest("POST", "/api/v1/clusters", string(invalidAuthConfig))
	code, _, body := result(handler, req)
	assert.Equal(t, 400, code, "invalid authentication configuration should be rejected. response: %s, %s", string(body))

	validAuthConfig, err := json.Marshal(models.Kluster{
		Name: "auth",
		Spec: models.KlusterSpec{
			AuthenticationConfiguration: models.AuthenticationConfiguration(`
apiVersion: apiserver.config.k8s.io/v1beta1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: https://issuer1.example.com
    audiences:
    - audience1
    - audience2
    audienceMatchPolicy: MatchAny
  claimMappings:
    username:
      expression: 'claims.username'
    groups:
      expression: 'claims.groups'
`),
		},
	})
	assert.NoError(t, err, "failed to marshal authentication configuration")
	req = createRequest("POST", "/api/v1/clusters", string(validAuthConfig))
	code, _, body = result(handler, req)
	assert.Equal(t, 201, code, "valid authentication configuration should be accepted. response: %s", string(body))

	emptyAuthConfig, err := json.Marshal(models.Kluster{
		Name: "emptyauth",
		Spec: models.KlusterSpec{
			ClusterCIDR: conv.Pointer("100.101.0.0/16"),
			AuthenticationConfiguration: models.AuthenticationConfiguration(`
apiVersion: apiserver.config.k8s.io/v1beta1
kind: AuthenticationConfiguration
`),
		},
	})
	assert.NoError(t, err, "failed to marshal authentication configuration")
	req = createRequest("POST", "/api/v1/clusters", string(emptyAuthConfig))
	code, _, body = result(handler, req)
	assert.Equal(t, 201, code, "empty authentication configuration is valid. response: %s", string(body))
}

func TestClusterShow(t *testing.T) {
	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{Name: "nase"},
	}
	handler, _, cancel := createTestHandler(t, &kluster)
	defer cancel()

	//Test Success
	req := createRequest("GET", "/api/v1/clusters/nase", "")
	code, _, body := result(handler, req)
	if !assert.Equal(t, 200, code) {
		return
	}
	var apiKluster models.Kluster
	assert.NoError(t, apiKluster.UnmarshalBinary(body), "Failed to parse response")
	assert.Equal(t, "nase", apiKluster.Name)

	//Test 404
	req = createRequest("GET", "/api/v1/clusters/doesnotexit", "")
	code, _, _ = result(handler, req)
	assert.Equal(t, 404, code)
}

func TestClusterList(t *testing.T) {
	kluster1 := &kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{Name: "nase"},
	}
	kluster2 := &kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "mund", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{Name: "mund"},
	}
	otherKluster := &kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "ohr", "other"),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": "other"},
		},
		Spec: models.KlusterSpec{Name: "ohr"},
	}
	handler, _, cancel := createTestHandler(t, kluster1, kluster2, otherKluster)
	defer cancel()

	//Test Success
	req := createRequest("GET", "/api/v1/clusters", "")
	code, _, body := result(handler, req)
	if !assert.Equal(t, 200, code) {
		return
	}
	var apiKlusters []models.Kluster
	assert.NoError(t, json.Unmarshal(body, &apiKlusters), "Failed to parse response")
	assert.ElementsMatch(t, []models.Kluster{
		{
			Name: "nase",
			Spec: models.KlusterSpec{Name: "nase"},
		},
		{
			Name: "mund",
			Spec: models.KlusterSpec{Name: "mund"},
		},
	}, apiKlusters)

}

func TestClusterUpdate(t *testing.T) {

	on := true
	off := false

	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{
			AdvertiseAddress: "0.0.0.0",
			ClusterCIDR:      conv.Pointer("1.1.1.1/24"),
			DNSAddress:       "2.2.2.254",
			DNSDomain:        "cluster.local",
			Name:             "nase",
			ServiceCIDR:      "2.2.2.2/24",
			Openstack: models.OpenstackSpec{
				LBSubnetID: "lbid",
				NetworkID:  "networkid",
				RouterID:   "routerid",
			},
			NodePools: []models.NodePool{
				{
					AvailabilityZone: "us-west-1a",
					Flavor:           "flavour",
					Image:            "image",
					Name:             "poolname",
					Size:             2,
					Config: &models.NodePoolConfig{
						AllowReboot:  &off,
						AllowReplace: &off,
					},
				},
			},
		},
		Status: models.KlusterStatus{
			Phase:   models.KlusterPhaseRunning,
			Version: "someversion",
		},
	}
	handler, _, cancel := createTestHandler(t, &kluster)
	defer cancel()
	updateObject := models.Kluster{
		Name: "mund",
		Spec: models.KlusterSpec{
			AdvertiseAddress: "7.7.7.7",
			ClusterCIDR:      conv.Pointer("8.8.8.8/24"),
			DNSAddress:       "9.9.9.9",
			DNSDomain:        "changed",
			ServiceCIDR:      "9.9.9.9/24",
			Openstack: models.OpenstackSpec{
				LBSubnetID: "changed",
				NetworkID:  "changed",
				RouterID:   "changed",
			},
			NodePools: []models.NodePool{
				{
					AvailabilityZone: "us-west-1a",
					Flavor:           "flavour",
					Image:            "image",
					Name:             "poolname",
					Size:             5,
					Config: &models.NodePoolConfig{
						AllowReboot:  &on,
						AllowReplace: &on,
					},
				},
				{
					AvailabilityZone: "us-east-1a",
					Flavor:           "newflavour",
					Image:            "newimage",
					Name:             "newpoolname",
					Size:             3,
					Config: &models.NodePoolConfig{
						AllowReboot:  &on,
						AllowReplace: &on,
					},
				},
			},
		},
		Status: models.KlusterStatus{
			Phase:   models.KlusterPhaseTerminating,
			Version: "changed",
		},
	}
	jsonPayload, err := updateObject.MarshalBinary()
	assert.NoError(t, err, "marshaling update payload failed")
	req := createRequest("PUT", "/api/v1/clusters/nase", string(jsonPayload))
	code, _, body := result(handler, req)
	if !assert.Equal(t, 200, code) {
		fmt.Printf("%s", string(body))
		return
	}
	var apiResponse models.Kluster
	assert.NoError(t, apiResponse.UnmarshalBinary(body), "Failed to parse response")

	//assert fields are immutable
	assert.Equal(t, "nase", apiResponse.Name)
	assert.Equal(t, "0.0.0.0", apiResponse.Spec.AdvertiseAddress)
	assert.Equal(t, "1.1.1.1/24", *apiResponse.Spec.ClusterCIDR)
	assert.Equal(t, "2.2.2.254", apiResponse.Spec.DNSAddress)
	assert.Equal(t, "cluster.local", apiResponse.Spec.DNSDomain)
	assert.Equal(t, "nase", apiResponse.Spec.Name)
	assert.Equal(t, "2.2.2.2/24", apiResponse.Spec.ServiceCIDR)
	assert.Equal(t, "lbid", apiResponse.Spec.Openstack.LBSubnetID)
	assert.Equal(t, "networkid", apiResponse.Spec.Openstack.NetworkID)
	assert.Equal(t, "routerid", apiResponse.Spec.Openstack.RouterID)
	assert.Equal(t, models.KlusterPhaseRunning, apiResponse.Status.Phase)
	assert.Equal(t, "someversion", apiResponse.Status.Version)

	//assert nodepool was updated
	assert.Equal(t, updateObject.Spec.NodePools, apiResponse.Spec.NodePools)
}

func TestVersionUpdate(t *testing.T) {

	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{
			Version: "1.10.1",
		},
		Status: models.KlusterStatus{
			ApiserverVersion: "1.10.1",
		},
	}

	cases := []struct {
		Version       string
		Phase         models.KlusterPhase
		ExpectSuccess bool
	}{
		{"1.10.1", models.KlusterPhaseRunning, true},
		{"1.10.0", models.KlusterPhaseRunning, true},
		{"1.10.2", models.KlusterPhaseRunning, true},
		{"1.10.3", models.KlusterPhaseUpgrading, false},
		{"1.11.2", models.KlusterPhaseRunning, true},
		{"1.9.2", models.KlusterPhaseRunning, false},
		{"1.12.2", models.KlusterPhaseRunning, false},
		{"2.0.0", models.KlusterPhaseRunning, false},
	}

	for _, c := range cases {
		k := kluster.DeepCopy()
		k.Status.Phase = c.Phase
		handler, _, cancel := createTestHandler(t, k)
		defer cancel()
		updateObject := models.Kluster{
			Name: "nase",
			Spec: models.KlusterSpec{
				Version: c.Version,
			},
		}
		jsonPayload, err := updateObject.MarshalBinary()
		if !assert.NoError(t, err, "marshaling update payload failed version %s", c.Version) {
			continue
		}
		req := createRequest("PUT", "/api/v1/clusters/nase", string(jsonPayload))
		code, _, body := result(handler, req)

		if c.ExpectSuccess {
			if assert.Equal(t, 200, code, "Update to version %s should be accepted. Response: %d,  %s", c.Version, code, string(body)) {
				var apiResponse models.Kluster
				assert.NoError(t, apiResponse.UnmarshalBinary(body), "Failed to parse response for version %s", c.Version)
				assert.Equal(t, c.Version, apiResponse.Spec.Version, "Update to version %s failed", c.Version)
			}
		} else {
			assert.Equal(t, 400, code, "Update to version %s should be rejected. Response: %d, %s", c.Version, code, string(body))
		}
	}
}

func TestClusterBootstrapConfig(t *testing.T) {

	kluster := &kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: NAMESPACE,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{
			Version:   "1.10.1",
			DNSDomain: "example.com",
		},
		Status: models.KlusterStatus{
			ApiserverVersion: "1.10.1",
		},
	}

	handler, rt, cancel := createTestHandler(t, kluster)
	defer cancel()
	_, err := util.EnsureKlusterSecret(rt.Kubernetes, kluster)
	if !assert.NoError(t, err, "failed to ensure kluster secret") {
		return
	}

	req := createRequest("GET", "/api/v1/clusters/nase/bootstrap", "")
	code, _, body := result(handler, req)

	require.Equal(t, 200, code, string(body))

	//I don't want to vendor https://github.com/kubernetes/kubelet sp I'm just checking if the yaml parses correctly

	var apiResponse models.BootstrapConfig
	require.NoError(t, apiResponse.UnmarshalBinary(body), "failed to parse api response")

	type fakeKubeletConfig struct {
		Kind               string `yaml:"kind"`
		APIVersion         string `yaml:"apiVersion"`
		RotateCertificates bool   `yaml:"rotateCertificates,omitempty"`
		ClusterDomain      string `yaml:"clusterDomain,omitempty"`
	}

	var config fakeKubeletConfig

	require.NoError(t, yaml.Unmarshal([]byte(apiResponse.Config), &config))
	assert.Equal(t, fakeKubeletConfig{
		Kind:               "KubeletConfiguration",
		APIVersion:         "kubelet.config.k8s.io/v1beta1",
		ClusterDomain:      "example.com",
		RotateCertificates: true,
	}, config)
}
