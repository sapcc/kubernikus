package handlers

import (
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewListClusters(rt *api.Runtime) operations.ListClustersHandler {
	return &listClusters{rt}
}

type listClusters struct {
	*api.Runtime
}

func (d *listClusters) Handle(params operations.ListClustersParams, principal *models.Principal) middleware.Responder {
	klusterList, err := d.Klusters.List(accountSelector(principal))

	if err != nil {
		return NewErrorResponse(&operations.ListClustersDefault{}, 500, err.Error())
	}

	clusters := make([]*models.Kluster, 0, len(klusterList))
	for _, kluster := range klusterList {
		clusters = append(clusters, klusterFromCRD(kluster))
	}
	return operations.NewListClustersOK().WithPayload(clusters)
}
