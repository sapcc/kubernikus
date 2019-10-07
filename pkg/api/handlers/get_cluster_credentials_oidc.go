package handlers

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
)

func NewGetClusterCredentialsOIDC(rt *api.Runtime) operations.GetClusterCredentialsOIDCHandler {
	return &getClusterCredentialsOIDC{rt}
}

type getClusterCredentialsOIDC struct {
	*api.Runtime
}

func (d *getClusterCredentialsOIDC) Handle(params operations.GetClusterCredentialsOIDCParams, principal *models.Principal) middleware.Responder {

	kluster, err := d.Klusters.Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Kluster not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	if !kluster.Spec.Dex {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Dex not enabled, no OIDC credentials")
	}

	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	ingress, err := d.Runtime.Kubernetes.ExtensionsV1beta1().Ingresses(d.Namespace).Get(fmt.Sprintf("%s-dex", kluster.Name), meta_v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Ingress not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}
	idpHost := ""
	if len(ingress.Spec.Rules) == 0 {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "no rule found in ingress")
	} else {
		idpHost = ingress.Spec.Rules[0].Host
	}

	config := kubernetes.NewClientConfigV1OIDC(
		params.Name,
		fmt.Sprintf("oidc@%v", params.Name),
		kluster.Status.Apiserver,
		secret.DexClientSecret,
		fmt.Sprintf("https://%s", idpHost),
		[]byte(secret.TLSCACertificate),
	)

	kubeconfig, err := yaml.Marshal(config)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsOIDCDefault{}, 500, "Failed to generate YAML document: %s", err)
	}

	credentials := models.Credentials{
		Kubeconfig: string(kubeconfig),
	}

	return operations.NewGetClusterCredentialsOIDCOK().WithPayload(&credentials)
}
