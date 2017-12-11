package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/openstack/scoped"
)

func NewGetOpenstackMetadata(rt *api.Runtime) operations.GetOpenstackMetadataHandler {
	return &getOpenstackMetadata{rt}
}

type getOpenstackMetadata struct {
	*api.Runtime
}

func (d *getOpenstackMetadata) Handle(params operations.GetOpenstackMetadataParams, principal *models.Principal) middleware.Responder {
	tokenID := params.HTTPRequest.Header.Get("X-Auth-Token")

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: principal.AuthURL,
		TokenID:          tokenID,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectID: principal.Account,
		},
	}

	client, err := scoped.NewClient(authOptions, d.Logger)
	if err != nil {
		return NewErrorResponse(&operations.GetOpenstackMetadataDefault{}, 500, err.Error())
	}

	openstackMetadata, err := client.GetMetadata()
	if err != nil {
		return NewErrorResponse(&operations.GetOpenstackMetadataDefault{}, 500, err.Error())
	}

	return operations.NewGetOpenstackMetadataOK().WithPayload(openstackMetadata)
}
