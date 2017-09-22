package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func NewTerminateCluster(rt *api.Runtime) operations.TerminateClusterHandler {
	return &terminateCluster{rt}
}

type terminateCluster struct {
	*api.Runtime
}

func (d *terminateCluster) Handle(params operations.TerminateClusterParams, principal *models.Principal) middleware.Responder {

	_, err := editCluster(d.Kubernikus.Kubernikus().Klusters(d.Namespace), principal, params.Name, func(kluster *v1.Kluster) {
		kluster.Status.Kluster.State = v1.KlusterTerminating
		kluster.Status.Kluster.Message = "Cluster terminating"
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.TerminateClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.TerminateClusterDefault{}, 500, err.Error())

	}
	return operations.NewTerminateClusterAccepted()
}
