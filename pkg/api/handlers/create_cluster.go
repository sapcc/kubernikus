package handlers

import (
	"fmt"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewCreateCluster(rt *api.Runtime) operations.CreateClusterHandler {
	return &createCluster{rt: rt}
}

type createCluster struct {
	rt *api.Runtime
}

func (d *createCluster) Handle(params operations.CreateClusterParams, principal *models.Principal) middleware.Responder {
	kluster := &tprv1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", params.Body.Name, principal.Account),
			Labels:      map[string]string{"account": principal.Account},
			Annotations: map[string]string{"creator": principal.Name},
		},
		Spec: tprv1.KlusterSpec{
			Name: params.Body.Name,
		},
		Status: tprv1.KlusterStatus{
			State: tprv1.KlusterPending,
		},
	}

	if err := d.rt.Clients.TPRClient().Post().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).Body(kluster).Do().Error(); err != nil {
		glog.Errorf("Failed to create cluster: %s", err)
		if apierrors.IsAlreadyExists(err) {
			return operations.NewCreateClusterDefault(409).WithPayload(modelsError(err))
		}
		return operations.NewCreateClusterDefault(500).WithPayload(modelsError(err))
	}

	return operations.NewCreateClusterOK()
}
