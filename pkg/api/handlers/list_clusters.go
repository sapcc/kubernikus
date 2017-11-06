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
	klusterList, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).List(listOpts)

	if err != nil {
		return NewErrorResponse(&operations.ListClustersDefault{}, 500, err.Error())
	}

	clusters := make([]*models.Cluster, 0, len(klusterList.Items))
	for _, kluster := range klusterList.Items {
		clusters = append(clusters, clusterModelFromCRD(&kluster))
	}
	return operations.NewListClustersOK().WithPayload(clusters)
}
