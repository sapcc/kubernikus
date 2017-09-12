package handlers

import (
	"crypto/x509"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
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

	bundle, err := ground.NewBundle(clientCAKey, clientCACert)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to parse CA certificate: %s", err)
	}

	cert := bundle.Sign(ground.Config{
		Sign:         principal.Name,
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	config := kubernetes.NewClientConfigV1(
		params.Name,
		principal.Name,
		fmt.Sprintf("https://%s.kluster.staging.cloud.sap", qualifiedName(params.Name, principal.Account)),
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
