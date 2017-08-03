package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewTerminateCluster(rt *api.Runtime) operations.TerminateClusterHandler {
	return &terminateCluster{rt: rt}
}

type terminateCluster struct {
	rt *api.Runtime
}

func (d *terminateCluster) Handle(params operations.TerminateClusterParams, principal *models.Principal) middleware.Responder {

	if err := d.rt.Clients.TPRClient().Delete().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(params.Name).Do().Error(); err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewTerminateClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewTerminateClusterDefault(0).WithPayload(modelsError(err))
	}

	return operations.NewTerminateClusterOK()
}
