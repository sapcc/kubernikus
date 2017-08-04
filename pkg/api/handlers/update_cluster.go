package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func NewUpdateCluster(rt *api.Runtime) operations.UpdateClusterHandler {
	return &updateCluster{rt: rt}
}

type updateCluster struct {
	rt *api.Runtime
}

func (d *updateCluster) Handle(params operations.UpdateClusterParams, principal *models.Principal) middleware.Responder {

	_, err := editCluster(d.rt.Clients.TPRClient(), principal, params.Name, func(kluster *tprv1.Kluster) {
		//TODO: currently no field to update
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewUpdateClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewUpdateClusterDefault(0).WithPayload(modelsError(err))
	}
	return operations.NewUpdateClusterOK()
}
