package project

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

type LoggingClient struct {
	Client ProjectClient
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
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.GetMetadata()
}

func (c LoggingClient) GetProjectTags() (tags []string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "fetched project tags",
			"tags", fmt.Sprintf("%v", tags),
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.GetProjectTags()
}
