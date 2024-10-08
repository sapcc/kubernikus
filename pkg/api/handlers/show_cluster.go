package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewShowCluster(rt *api.Runtime) operations.ShowClusterHandler {
	return &showCluster{rt}
}

type showCluster struct {
	*api.Runtime
}

func (d *showCluster) Handle(params operations.ShowClusterParams, principal *models.Principal) middleware.Responder {
	name := qualifiedName(params.Name, principal.Account)
	kluster, err := d.Klusters.Klusters(d.Namespace).Get(name)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.ShowClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.ShowClusterDefault{}, 500, "%s", err)
	}

	return operations.NewShowClusterOK().WithPayload(klusterFromCRD(kluster))
}
