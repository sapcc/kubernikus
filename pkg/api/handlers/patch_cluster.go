package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/golang/glog"
	"fmt"
)

func NewPatchCluster(rt *api.Runtime) operations.PatchClusterHandler {
	return &patchCluster{rt: rt}
}

type patchCluster struct {
	rt *api.Runtime
}

func (d *patchCluster) Handle(params operations.PatchClusterParams, principal *models.Principal) middleware.Responder {
	var oldKluster tprv1.Kluster
	if err := d.rt.Clients.TPRClient().Get().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(qualifiedName(params.Name,principal.Account)).Do().Into(&oldKluster); err != nil {
		if apierrors.IsNotFound(err) {
			return operations.NewPatchClusterDefault(404).WithPayload(modelsError(err))
		}
		return operations.NewPatchClusterDefault(0).WithPayload(modelsError(err))
	}

	kluster := &tprv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", params.Body.Name, principal.Account),
			Labels:      map[string]string{"account": principal.Account},
			Annotations: map[string]string{"creator": principal.Name},
		},
		Spec: tprv1.KlusterSpec{
			Name:    params.Body.Name,
			Account: principal.Account,
		},
		Status: tprv1.KlusterStatus{
			State: oldKluster.Status.State,
		},
	}

	patchBytes, patchType, err := createPatch(&oldKluster,kluster)
	if err != nil {
		return operations.NewPatchClusterDefault(0).WithPayload(modelsError(err))
	}

	if err := d.rt.Clients.TPRClient().Patch(patchType).Body(patchBytes).Do().Error(); err != nil {
		glog.Errorf("Failed to patch %s/%s: %s",oldKluster.GetNamespace(),oldKluster.GetName(),err)
		return operations.NewPatchClusterDefault(0).WithPayload(modelsError(err))
	}

	return operations.NewPatchClusterOK()
}
