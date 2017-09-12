package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewListClusters(rt *api.Runtime) operations.ListClustersHandler {
	return &listClusters{rt}
}

type listClusters struct {
	*api.Runtime
}

func (d *listClusters) Handle(params operations.ListClustersParams, principal *models.Principal) middleware.Responder {
	listOpts := metav1.ListOptions{LabelSelector: accountSelector(principal).String()}
	clusterList, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).List(listOpts)

	if err != nil {
		return NewErrorResponse(&operations.ListClustersDefault{}, 500, err.Error())
	}

	clusters := make([]*models.Cluster, 0, len(clusterList.Items))
	for _, c := range clusterList.Items {
		clusters = append(clusters, &models.Cluster{Name: &c.Spec.Name, Status: string(c.Status.State)})
	}
	return operations.NewListClustersOK().WithPayload(clusters)
}
