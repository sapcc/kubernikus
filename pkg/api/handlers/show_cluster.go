package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewShowCluster(rt *api.Runtime) operations.ShowClusterHandler {
	return &showCluster{rt}
}

type showCluster struct {
	*api.Runtime
}

func (d *showCluster) Handle(params operations.ShowClusterParams, principal *models.Principal) middleware.Responder {
	name := qualifiedName(params.Name, principal.Account)
	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.ShowClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.ShowClusterDefault{}, 500, err.Error())
	}

	return operations.NewShowClusterOK().WithPayload(klusterFromCRD(kluster))
}
