package kluster

import (
	"time"

	"github.com/go-kit/kit/log"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

type LoggingClient struct {
	Client KlusterClient
	Logger log.Logger
}

func (c LoggingClient) DeleteNode(id string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "deleted node",
			"id", id,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.DeleteNode(id)
}

func (c LoggingClient) CreateNode(pool *models.NodePool, userData []byte) (id string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "created node",
			"id", id,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.CreateNode(pool, userData)
}

func (c LoggingClient) ListNodes(pool *models.NodePool) (nodes []Node, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "listed nodes",
			"count", len(nodes),
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.ListNodes(pool)
}
