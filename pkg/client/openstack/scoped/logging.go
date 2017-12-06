package scoped

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

type LoggingClient struct {
	Client Client
	Logger log.Logger
}

func (c LoggingClient) GetMetadata() (metadata *models.OpenstackMetadata, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "fetched metadata",
			"flavors", len(metadata.Flavors),
			"keypairs", len(metadata.KeyPairs),
			"routers", len(metadata.Routers),
			"security_groups", len(metadata.SecurityGroups),
			"took", time.Since(begin),
			"v", 1,
			"err", err,
		)
	}(time.Now())

	return c.Client.GetMetadata()
}

func (c LoggingClient) Authenticate(authOptions *tokens.AuthOptions) (err error) {
	defer func(begin time.Time) {
		v := 2
		if err != nil {
			v = 0
		}
		c.Logger.Log(
			"msg", "authenticated",
			"took", time.Since(begin),
			"v", v,
			"err", err,
		)
	}(time.Now())
	return c.Client.Authenticate(authOptions)
}
