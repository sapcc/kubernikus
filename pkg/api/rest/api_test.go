package rest

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	errors "github.com/go-openapi/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	apipkg "github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
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
		AuthURL: "http://identity.test/v3",
		ID:      "test",
		Name:    "Test Mc Dougle",
		Domain:  "TestDomain",
		Account: ACCOUNT,
		Roles:   []string{"member", "kubernetes_admin"},
	}, nil
}

func createTestHandler(t *testing.T) (http.Handler, *apipkg.Runtime) {
	swaggerSpec, err := spec.Spec()
	if err != nil {
		t.Fatal(err)
	}
	api := operations.NewKubernikusAPI(swaggerSpec)
	rt := &apipkg.Runtime{
		Namespace:  NAMESPACE,
		Kubernikus: kubernikusfake.NewSimpleClientset(),
		Kubernetes: fake.NewSimpleClientset(),
		Logger:     kitlog.NewNopLogger(),
	}
	if err := Configure(api, rt); err != nil {
		t.Fatal(err)
	}
	api.KeystoneAuth = mockAuth
	return configureAPI(api), rt
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
	body, _ := ioutil.ReadAll(response.Body)
	return response.StatusCode, response.Header, body
}

func TestCreateCluster(t *testing.T) {
	handler, rt := createTestHandler(t)
	req := createRequest("POST", "/api/v1/clusters", `{"name": "nase"}`)
	code, _, body := result(handler, req)
	if !assert.Equal(t, 201, code) {
		return
	}

	//Test create
	crd, err := rt.Kubernikus.KubernikusV1().Klusters(rt.Namespace).Get(fmt.Sprintf("%s-%s", "nase", ACCOUNT), metav1.GetOptions{})
	assert.NoError(t, err, "resource not persisted")
	assert.Equal(t, crd.Labels["account"], ACCOUNT)

	var kluster models.Kluster
	assert.NoError(t, kluster.UnmarshalBinary(body), "Failed to parse response")
	assert.Equal(t, "nase", kluster.Name)
	assert.Equal(t, "nase", kluster.Spec.Name)
	assert.Equal(t, models.KlusterPhasePending, kluster.Status.Phase)

	//Ensure authentication is required
	req = createRequest("POST", "/api/v1/clusters", `{"name": "nase2"}`)
	req.Header.Del("X-Auth-Token")
	code, _, _ = result(handler, req)
	assert.Equal(t, 401, code)

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

}

func TestClusterShow(t *testing.T) {
	handler, rt := createTestHandler(t)
	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: rt.Namespace,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{Name: "nase"},
	}

	rt.Kubernikus = kubernikusfake.NewSimpleClientset(&kluster)

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

func TestClusterUpdate(t *testing.T) {
	handler, rt := createTestHandler(t)
	kluster := kubernikusv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "nase", ACCOUNT),
			Namespace: rt.Namespace,
			Labels:    map[string]string{"account": ACCOUNT},
		},
		Spec: models.KlusterSpec{
			AdvertiseAddress: "0.0.0.0",
			ClusterCIDR:      "1.1.1.1/24",
			DNSAddress:       "2.2.2.254",
			DNSDomain:        "cluster.local",
			Name:             "nase",
			ServiceCIDR:      "2.2.2.2/24",
			Openstack: models.OpenstackSpec{
				LBSubnetID: "lbid",
				NetworkID:  "networkid",
				ProjectID:  ACCOUNT,
				RouterID:   "routerid",
			},
			NodePools: []models.NodePool{
				{
					Flavor: "flavour",
					Image:  "image",
					Name:   "poolname",
					Size:   2,
				},
			},
		},
		Status: models.KlusterStatus{
			Phase:   models.KlusterPhaseRunning,
			Version: "someversion",
		},
	}
	rt.Kubernikus = kubernikusfake.NewSimpleClientset(&kluster)
	updateObject := models.Kluster{
		Name: "mund",
		Spec: models.KlusterSpec{
			AdvertiseAddress: "7.7.7.7",
			ClusterCIDR:      "8.8.8.8/24",
			DNSAddress:       "9.9.9.9.254",
			DNSDomain:        "changed",
			Name:             "mund",
			ServiceCIDR:      "9.9.9.9/24",
			Openstack: models.OpenstackSpec{
				LBSubnetID: "changed",
				NetworkID:  "changed",
				ProjectID:  "changed",
				RouterID:   "changed",
			},
			NodePools: []models.NodePool{
				{
					Flavor: "newflavour",
					Image:  "newimage",
					Name:   "newpoolname",
					Size:   3,
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
		return
	}
	var apiResponse models.Kluster
	assert.NoError(t, apiResponse.UnmarshalBinary(body), "Failed to parse response")

	//assert fields are immutable
	assert.Equal(t, "nase", apiResponse.Name)
	assert.Equal(t, "0.0.0.0", apiResponse.Spec.AdvertiseAddress)
	assert.Equal(t, "1.1.1.1/24", apiResponse.Spec.ClusterCIDR)
	assert.Equal(t, "2.2.2.254", apiResponse.Spec.DNSAddress)
	assert.Equal(t, "cluster.local", apiResponse.Spec.DNSDomain)
	assert.Equal(t, "nase", apiResponse.Spec.Name)
	assert.Equal(t, "2.2.2.2/24", apiResponse.Spec.ServiceCIDR)
	assert.Equal(t, "lbid", apiResponse.Spec.Openstack.LBSubnetID)
	assert.Equal(t, "networkid", apiResponse.Spec.Openstack.NetworkID)
	assert.Equal(t, ACCOUNT, apiResponse.Spec.Openstack.ProjectID)
	assert.Equal(t, "routerid", apiResponse.Spec.Openstack.RouterID)
	assert.Equal(t, models.KlusterPhaseRunning, apiResponse.Status.Phase)
	assert.Equal(t, "someversion", apiResponse.Status.Version)

	//assert nodepool was updated
	assert.Equal(t, updateObject.Spec.NodePools, apiResponse.Spec.NodePools)

}
