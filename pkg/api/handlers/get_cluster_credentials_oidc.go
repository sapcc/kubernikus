package handlers

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

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
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "%s", err)
	}

	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "%s", err)
	}

	secretID := secret.DexClientSecret
	clientID := "kubernetes"
	issuerURL := ""
	if kluster.Status.Apiserver != "" {
		apiURL := kluster.Status.Apiserver
		issuerURL = strings.Replace(apiURL, kluster.GetName(), fmt.Sprintf("auth-%s.ingress", kluster.GetName()), -1)
	}

	if !swag.BoolValue(kluster.Spec.Dex) {
		if secret.ExtraValues != "" {
			var oidcValues struct {
				API struct {
					OIDC struct {
						ClientID  string `yaml:"clientID,omitempty"`
						SecretID  string `yaml:"secretID,omitempty"`
						IssuerURL string `yaml:"issuerURL,omitempty"`
					} `yaml:"oidc"`
				} `yaml:"api"`
			}
			if err := yaml.Unmarshal([]byte(secret.ExtraValues), &oidcValues); err == nil && oidcValues.API.OIDC.IssuerURL != "" {
				issuerURL = oidcValues.API.OIDC.IssuerURL
				clientID = oidcValues.API.OIDC.ClientID
				secretID = oidcValues.API.OIDC.SecretID
			} else {
				return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Dex not enabled, no OIDC credentials")
			}
		} else {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Dex not enabled, no OIDC credentials")
		}
	}

	config := kubernetes.NewClientConfigV1OIDC(
		params.Name,
		fmt.Sprintf("oidc@%v", params.Name),
		kluster.Status.Apiserver,
		clientID,
		secretID,
		issuerURL,
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
