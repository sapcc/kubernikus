package handlers

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

func NewGetClusterCredentials(rt *api.Runtime) operations.GetClusterCredentialsHandler {
	return &getClusterCredentials{rt}
}

type getClusterCredentials struct {
	*api.Runtime
}

func (d *getClusterCredentials) Handle(params operations.GetClusterCredentialsParams, principal *models.Principal) middleware.Responder {

	secret, err := d.Kubernetes.CoreV1().Secrets(d.Namespace).Get(qualifiedName(params.Name, principal.Account), v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account), v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	clientCAKey, ok := secret.Data["apiserver-clients-ca-key.pem"]
	if !ok {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Clients CA key not found")
	}
	clientCACert, ok := secret.Data["apiserver-clients-ca.pem"]
	if !ok {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Clients CA certificate not found")
	}
	serverCACert, ok := secret.Data["tls-ca.pem"]
	if !ok {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Server CA certificate not found")
	}

	bundle, err := util.NewBundle(clientCAKey, clientCACert)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to parse CA certificate: %s", err)
	}

	var organizations []string
	for _, role := range principal.Roles {
		organizations = append(organizations, "os:"+role)
	}

	cert := bundle.Sign(util.Config{
		Sign:         fmt.Sprintf("%s@%s", principal.Name, principal.Domain),
		Organization: organizations,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ValidFor:     24 * time.Hour,
	})
	config := kubernetes.NewClientConfigV1(
		params.Name,
		fmt.Sprintf("%v@%v", principal.Name, params.Name),
		kluster.Status.Apiserver,
		certutil.EncodePrivateKeyPEM(cert.PrivateKey),
		certutil.EncodeCertPEM(cert.Certificate),
		serverCACert,
	)

	kubeconfig, err := yaml.Marshal(config)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to generate YAML document: %s", err)
	}

	credentials := models.Credentials{
		Kubeconfig: string(kubeconfig),
	}

	return operations.NewGetClusterCredentialsOK().WithPayload(&credentials)
}
