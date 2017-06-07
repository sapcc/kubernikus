package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func ListAPIVersions(params operations.ListAPIVersionsParams) middleware.Responder {
	r := operations.NewListAPIVersionsOK()
	r.Payload = &models.APIVersions{Versions: []string{"v1"}}
	return r
}
