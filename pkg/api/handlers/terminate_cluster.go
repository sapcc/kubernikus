package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	"github.com/golang/glog"
)

func NewTerminateCluster(rt *api.Runtime) operations.TerminateClusterHandler {
	return &terminateCluster{rt: rt}
}

type terminateCluster struct {
	rt *api.Runtime
}

func (d *terminateCluster) Handle(params operations.TerminateClusterParams, principal *models.Principal) middleware.Responder {
	var oldKluster tprv1.Kluster
	if err := d.rt.Clients.TPRClient().Get().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(qualifiedName(params.Name,principal.Account)).Do().Into(&oldKluster); err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewTerminateClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewTerminateClusterDefault(0).WithPayload(modelsError(err))
	}

	copy, err := d.rt.Clients.TPRScheme().Copy(&oldKluster)
	if err != nil {

	}
	cpKluster := copy.(*tprv1.Kluster)
	cpKluster.Status.State = tprv1.KlusterTerminating

	patchBytes, patchType, err := createPatch(&oldKluster,cpKluster)
	if err != nil {
		return operations.NewTerminateClusterDefault(0).WithPayload(modelsError(err))
	}

	if err := d.rt.Clients.TPRClient().Patch(patchType).Body(patchBytes).Do().Error(); err != nil {
		glog.Errorf("Failed to patch %s/%s: %s",oldKluster.GetNamespace(),oldKluster.GetName(),err)
		return operations.NewTerminateClusterDefault(0).WithPayload(modelsError(err))
	}

	return operations.NewTerminateClusterOK()
}
