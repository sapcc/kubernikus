package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewShowCluster(rt *api.Runtime) operations.ShowClusterHandler {
	return &showCluster{rt: rt}
}

type showCluster struct {
	rt *api.Runtime
}

func (d *showCluster) Handle(params operations.ShowClusterParams, principal *models.Principal) middleware.Responder {
	var tprCluster tprv1.Kluster
	if err := d.rt.Clients.TPRClient().Get().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(params.Name).Do().Into(&tprCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewShowClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewShowClusterDefault(0).WithPayload(modelsError(err))
	}
	c := models.Cluster{
		Name:   tprCluster.Name,
		Status: string(tprCluster.Status.State),
	}

	return operations.NewShowClusterOK().WithPayload(&c)
}
