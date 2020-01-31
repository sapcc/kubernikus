package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
)

func TestKubernikusContext(t *testing.T) {

	kluster := &v1.Kluster{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"account": "12345678"}}, Spec: models.KlusterSpec{AdvertiseAddress: "1.1.1.1", ServiceCIDR: "192.168.0.0/24"}}
	certs := new(v1.Certificates)

	factory := util.NewCertificateFactory(kluster, certs, "test.local")
	_, err := factory.Ensure()
	require.NoError(t, err)
	bundle, err := factory.UserCert(&models.Principal{Name: "exampleuser", Domain: "exampledomain", AuthURL: "http://auth.url"}, "http://kubernikus.url")
	require.NoError(t, err)

	config := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test": {CertificateAuthorityData: []byte(certs.ApiserverClientsCACertifcate)},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test": {ClientCertificateData: certutil.EncodeCertPEM(bundle.Certificate)},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test": {Cluster: "test", AuthInfo: "test"},
		},
	}

	ctx := KubernikusContext{Config: config, context: "test"}
	b, err := ctx.IsKubernikusContext()
	require.NoError(t, err, "IsKubernikusContext shouldn't error")
	require.True(t, b, "IsKubernikusContext should be true")
	b, err = ctx.UserCertificateValid()
	assert.NoError(t, err, "UserCertificateExpired shouldn't error")
	assert.True(t, b, "UserCertificateExpired should be true")

	cases := []struct {
		Name         string
		Func         func() (string, error)
		ResultString string
	}{
		{Name: "AuthURL", Func: ctx.AuthURL, ResultString: "http://auth.url"},
		{Name: "Username", Func: ctx.Username, ResultString: "exampleuser"},
		{Name: "UserDomainame", Func: ctx.UserDomainname, ResultString: "exampledomain"},
		{Name: "ProjectID", Func: ctx.ProjectID, ResultString: "12345678"},
		{Name: "KubernikusURL", Func: ctx.KubernikusURL, ResultString: "http://kubernikus.url"},
	}

	for _, c := range cases {
		result, err := c.Func()
		if assert.NoError(t, err, "Expected %s to not return an error", c.Name) {
			assert.Equal(t, c.ResultString, result, "Expected %s to return %s", c.Name, c.ResultString)
		}
	}

}
