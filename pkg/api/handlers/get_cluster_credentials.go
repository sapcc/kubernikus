package handlers

import (
	"crypto/x509"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	certutil "k8s.io/client-go/util/cert"
)

func NewGetClusterCredentials(rt *api.Runtime) operations.GetClusterCredentialsHandler {
	return &getClusterCredentials{rt: rt}
}

type getClusterCredentials struct {
	rt *api.Runtime
}

func (d *getClusterCredentials) Handle(params operations.GetClusterCredentialsParams, principal *models.Principal) middleware.Responder {

	secret, err := d.rt.Clients.Kubernetes.CoreV1().Secrets("kubernikus").Get(qualifiedName(params.Name, principal.Account), v1.GetOptions{})
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
	config := clientcmdapiv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: params.Name,
		Clusters: []clientcmdapiv1.NamedCluster{
			clientcmdapiv1.NamedCluster{
				Name: params.Name,
				Cluster: clientcmdapiv1.Cluster{
					Server: fmt.Sprintf("https://%s.kluster.staging.cloud.sap", qualifiedName(params.Name, principal.Account)),
					CertificateAuthorityData: serverCACert,
				},
			},
		},
		Contexts: []clientcmdapiv1.NamedContext{
			clientcmdapiv1.NamedContext{
				Name: params.Name,
				Context: clientcmdapiv1.Context{
					Cluster:  params.Name,
					AuthInfo: principal.Name,
				},
			},
		},
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			clientcmdapiv1.NamedAuthInfo{
				Name: principal.Name,
				AuthInfo: clientcmdapiv1.AuthInfo{
					ClientCertificateData: certutil.EncodeCertPEM(cert.Certificate),
					ClientKeyData:         certutil.EncodePrivateKeyPEM(cert.PrivateKey),
				},
			},
		},
	}

	kubeconfig, err := yaml.Marshal(config)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to generate YAML document: %s", err)
	}

	credentials := models.Credentials{
		Kubeconfig: string(kubeconfig),
	}

	return operations.NewGetClusterCredentialsOK().WithPayload(&credentials)
}
