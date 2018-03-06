package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func NewUpdateCluster(rt *api.Runtime) operations.UpdateClusterHandler {
	return &updateCluster{rt}
}

type updateCluster struct {
	*api.Runtime
}

func (d *updateCluster) Handle(params operations.UpdateClusterParams, principal *models.Principal) middleware.Responder {

	kluster, err := editCluster(d.Kubernikus.Kubernikus().Klusters(d.Namespace), principal, params.Name, func(kluster *v1.Kluster) {
		nodePools := params.Body.Spec.NodePools
		//set default image
		for i, pool := range nodePools {
			if pool.Image == "" {
				nodePools[i].Image = DEFAULT_IMAGE
			}
		}
		// Update nodepool
		kluster.Spec.NodePools = nodePools
		kluster.Spec.SSHPublicKey = params.Body.Spec.SSHPublicKey

		if params.Body.Spec.Openstack.SecurityGroupName != "" {
			kluster.Spec.Openstack.SecurityGroupName = params.Body.Spec.Openstack.SecurityGroupName
		}
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.UpdateClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
	}
	return operations.NewUpdateClusterOK().WithPayload(klusterFromCRD(kluster))
}
