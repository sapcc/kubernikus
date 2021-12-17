package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/helm"
)

func NewGetClusterValues(rt *api.Runtime) operations.GetClusterValuesHandler {
	return &getClusterValues{Runtime: rt}
}

type getClusterValues struct {
	*api.Runtime
}

func (d *getClusterValues) Handle(params operations.GetClusterValuesParams, principal *models.Principal) middleware.Responder {

	//This is an admin-only api, the account is passed via parameters
	kluster, err := d.Kubernikus.KubernikusV1().Klusters(d.Namespace).Get(qualifiedName(params.Name, params.Account), meta_v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Kluster not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to retrieve cluster: %s", err)
	}
	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to retrieve cluster secret: %s", err)
	}

	accessMode, err := util.PVAccessMode(d.Kubernetes, kluster)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Couldn't determine access mode for pvc: %s", err)
	}

	values, err := helm.KlusterToHelmValues(kluster, secret, kluster.Spec.Version, d.Images, accessMode)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to generate helm values: %s", err)
	}

	yamlData, err := yaml.Marshal(&values)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "Failed to marshal helm values: %s", err)
	}
	payload := &models.GetClusterValuesOKBody{Values: string(yamlData)}

	return operations.NewGetClusterValuesOK().WithPayload(payload)
}
