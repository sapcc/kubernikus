package handlers

import (
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewGetOpenstackMetadata(rt *api.Runtime) operations.GetOpenstackMetadataHandler {
	return &getOpenstackMetadata{rt}
}

type getOpenstackMetadata struct {
	*api.Runtime
}

func (d *getOpenstackMetadata) Handle(params operations.GetOpenstackMetadataParams, principal *models.Principal) middleware.Responder {
	openstackMetadata, err := fetchOpenstackMetadata(params.HTTPRequest, principal)
	if err != nil {
		return NewErrorResponse(&operations.GetOpenstackMetadataDefault{}, 500, "%s", err)
	}

	return operations.NewGetOpenstackMetadataOK().WithPayload(openstackMetadata)
}
