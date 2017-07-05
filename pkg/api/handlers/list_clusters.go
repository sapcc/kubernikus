package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewListClusters(rt *api.Runtime) operations.ListClustersHandler {
	return &listClusters{rt: rt}
}

type listClusters struct {
	rt *api.Runtime
}

func (d *listClusters) Handle(params operations.ListClustersParams, principal *models.Principal) middleware.Responder {
	clusterList := tprv1.KlusterList{}
	if err := d.rt.TPRClient.Get().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Do().Into(&clusterList); err != nil {
		return operations.NewListClustersDefault(500).WithPayload(modelsError(err))
	}

	clusters := make([]*models.Cluster, 0, len(clusterList.Items))
	for _, c := range clusterList.Items {
		clusters = append(clusters, &models.Cluster{Name: c.Spec.Name, Status: string(c.Status.State)})
	}
	return operations.NewListClustersOK().WithPayload(clusters)
}
