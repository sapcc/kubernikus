package handlers

import (
	"encoding/base64"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/util"
)

func NewGetClusterKubeadmSecret(rt *api.Runtime) operations.GetClusterKubeadmSecretHandler {
	return &getClusterKubeadmSecret{rt}
}

type getClusterKubeadmSecret struct {
	*api.Runtime
}

func (d *getClusterKubeadmSecret) Handle(params operations.GetClusterKubeadmSecretParams, principal *models.Principal) middleware.Responder {

	kluster, err := d.Klusters.Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterKubeadmSecretDefault{}, 404, "Kluster not found")
		}
		return NewErrorResponse(&operations.GetClusterKubeadmSecretDefault{}, 500, "%s", err)
	}

	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "%s", err)
	}

	kadmSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-ca", kluster.Name),
		},
		StringData: map[string]string{
			"tls.crt": base64.StdEncoding.EncodeToString([]byte(secret.TLSCACertificate)),
			"tls.key": base64.StdEncoding.EncodeToString([]byte(secret.TLSCAPrivateKey)),
		},
	}

	secretYaml, err := yaml.Marshal(kadmSecret)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsKubeadmDefault{}, 500, "Failed to generate YAML document: %s", err)
	}

	secretData := models.KubeadmSecret{
		Secret: string(secretYaml),
	}

	return operations.NewGetClusterCredentialsKubeadmOK().WithPayload(&secretData)
}
