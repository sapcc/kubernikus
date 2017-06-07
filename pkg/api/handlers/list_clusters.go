package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func ListClusters(params operations.ListClustersParams, principal *models.Principal) middleware.Responder {
	return operations.NewListClustersOK().WithPayload(
		[]*models.Cluster{&models.Cluster{Name: "test123"}},
	)
}
