package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func NewTerminateCluster(rt *api.Runtime) operations.TerminateClusterHandler {
	return &terminateCluster{rt: rt}
}

type terminateCluster struct {
	rt *api.Runtime
}

func (d *terminateCluster) Handle(params operations.TerminateClusterParams, principal *models.Principal) middleware.Responder {

	_, err := editCluster(d.rt.Clients.TPRClient(), principal, params.Name, func(kluster *tprv1.Kluster) {
		kluster.Status.State = tprv1.KlusterTerminating
		kluster.Status.Message = "Cluster terminating"
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.TerminateClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.TerminateClusterDefault{}, 500, err.Error())

	}
	return operations.NewTerminateClusterOK()
}
