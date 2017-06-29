package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewListAPIVersions(rt *api.Runtime) operations.ListAPIVersionsHandler {
	return &listAPIVersions{rt: rt}
}

type listAPIVersions struct {
	rt *api.Runtime
}

func (d *listAPIVersions) Handle(params operations.ListAPIVersionsParams) middleware.Responder {
	return operations.NewListAPIVersionsOK().WithPayload(
		models.APIVersions{Versions: []string{"v1"}},
	)
}
