package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewDeleteCluster(rt *api.Runtime) operations.DeleteClusterHandler {
	return &deleteCluster{rt: rt}
}

type deleteCluster struct {
	rt *api.Runtime
}

func (d *deleteCluster) Handle(params operations.DeleteClusterParams, principal *models.Principal) middleware.Responder {
	if err := d.rt.Clients.TPRClient().Delete().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(params.Name).Do().Error(); err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewDeleteClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewDeleteClusterDefault(0).WithPayload(modelsError(err))
	}

	return operations.NewDeleteClusterOK()
}
