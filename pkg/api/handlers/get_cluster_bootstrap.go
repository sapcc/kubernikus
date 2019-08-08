package handlers

import (
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/bootstraptoken"
)

func NewGetBootstrapConfig(rt *api.Runtime) operations.GetBootstrapConfigHandler {
	return &getBootstrapConfig{rt}
}

type getBootstrapConfig struct {
	*api.Runtime
}

func (d *getBootstrapConfig) Handle(params operations.GetBootstrapConfigParams, principal *models.Principal) middleware.Responder {

	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account), meta_v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 404, "Kluster not found")
		}
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, err.Error())
	}

	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, err.Error())
	}

	client, err := d.KlusterClientFactory.ClientFor(kluster)
	if err != nil {
		return NewErrorResponse(&operations.GetBoostrapConfigDefault{}, 500, "Failed to create cluster client: %s", err)
	}

	token, tokenSecret, err := bootstraptoken.GenerateBootstrapToken()
	if err != nil {
		return NewErrorResponse(&operations.GetBoostrapConfigDefault{}, 500, "Failed to generate bootstrap token: %s", err)
	}
	if _, err := client.CoreV1().Secrets(tokenSecret.Namespace).Create(tokenSecret); err != nil {
		return NewErrorResponse(&operations.GetBoostrapConfigDefault{}, 500, "Failed to store bootstrap token: %s", err)
	}

	config := kubernetes.NewClientConfigV1(
		params.Name,
		"kubelet-bootstrap",
		kluster.Status.Apiserver,
		nil,
		nil,
		[]byte(secret.TLSCACertificate),
		token,
	)

	kubeconfig, err := yaml.Marshal(config)
	if err != nil {
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to generate YAML document: %s", err)
	}

	credentials := models.BootstrapConfig{
		Kubeconfig:       string(kubeconfig),
		KubeletClientsCA: secret.KubeletClientsCACertificate,
	}

	return operations.NewGetBootstrapConfigOK().WithPayload(&credentials)
}
