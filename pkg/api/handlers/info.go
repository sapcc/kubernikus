package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/version"
)

func NewInfo(rt *api.Runtime) operations.InfoHandler {
	return &info{rt}
}

type info struct {
	*api.Runtime
}

func (d *info) Handle(params operations.InfoParams) middleware.Responder {

	info := &models.Info{
		Binaries: []*models.InfoBinariesItems0{
			{
				Name: "kubernikusctl",
				Links: []*models.InfoBinariesItems0LinksItems0{
					{
						Platform: "darwin",
						Arch:     "amd64",
						Link:     "static/binaries/darwin/amd64/kubernikusctl",
					},
					{
						Platform: "linux",
						Arch:     "amd64",
						Link:     "static/binaries/linux/x86/kubernikusctl",
					},
				},
			},
		},
		Version: version.VERSION,
	}

	return operations.NewInfoOK().WithPayload(info)
}
