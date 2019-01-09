package handlers

import (
	"fmt"

	"github.com/databus23/requestutil"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
)

func NewGetClusterCredentials(rt *api.Runtime) operations.GetClusterCredentialsHandler {
	return &getClusterCredentials{rt}
}

type getClusterCredentials struct {
	*api.Runtime
}

func (d *getClusterCredentials) Handle(params operations.GetClusterCredentialsParams, principal *models.Principal) middleware.Responder {

	secret, err := d.Kubernetes.CoreV1().Secrets(d.Namespace).Get(qualifiedName(params.Name, principal.Account), meta_v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account), meta_v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	klusterSecret, err := v1.NewSecret(secret)
	factory := util.NewCertificateFactory(kluster, &klusterSecret.Certificates, "")

	var organizations []string
	for _, role := range principal.Roles {
		organizations = append(organizations, "os:"+role)
	}

	cert, err := factory.UserCert(principal, fmt.Sprintf("%s://%s", requestutil.Scheme(params.HTTPRequest), requestutil.HostWithPort(params.HTTPRequest)))
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to issue cert: %s", err)
	}
	config := kubernetes.NewClientConfigV1(
		params.Name,
		fmt.Sprintf("%v@%v", principal.Name, params.Name),
		kluster.Status.Apiserver,
		certutil.EncodePrivateKeyPEM(cert.PrivateKey),
		certutil.EncodeCertPEM(cert.Certificate),
		[]byte(klusterSecret.TLSCACertificate),
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
