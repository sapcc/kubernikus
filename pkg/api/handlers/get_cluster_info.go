package handlers

import (
	"fmt"

	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewGetClusterInfo(rt *api.Runtime) operations.GetClusterInfoHandler {
	return &getClusterInfo{rt}
}

type getClusterInfo struct {
	*api.Runtime
}

func (d *getClusterInfo) Handle(params operations.GetClusterInfoParams, principal *models.Principal) middleware.Responder {
	info := &models.KlusterInfo{
		SetupCommand: createSetupCommand(principal),
		Binaries: []models.Binaries{
			{
				Name: "kubernikusctl",
				Links: []models.Link{
					{
						Platform: "darwin",
						Link:     "static/binaries/darwin/amd64/kubernikusctl",
					},
					{
						Platform: "linux",
						Link:     "static/binaries/linux/amd64/kubernikusctl",
					},
				},
			},
		},
	}
	return operations.NewGetClusterInfoOK().WithPayload(info)
}

func createSetupCommand(principal *models.Principal) string {
	userName := principal.Name
	userDomainName := principal.Domain
	projectId := principal.Account
	authUrl := principal.AuthURL

	return fmt.Sprintf("kubernikusctl auth init --username %v --user-domain-name %v --project-id %v --auth-url %v", userName, userDomainName, projectId, authUrl)
}
