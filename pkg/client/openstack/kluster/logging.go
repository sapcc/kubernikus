package kluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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

func (c LoggingClient) RebootNode(id string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "rebooted node",
			"id", id,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.RebootNode(id)
}

func (c LoggingClient) CreateNode(kluster *v1.Kluster, pool *models.NodePool, nodeName string, userData []byte) (id string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "created node",
			"name", nodeName,
			"id", id,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.CreateNode(kluster, pool, nodeName, userData)
}

func (c LoggingClient) ListNodes(kluster *v1.Kluster, pool *models.NodePool) (nodes []Node, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "listed nodes",
			"count", len(nodes),
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.ListNodes(kluster, pool)
}

func (c LoggingClient) SetSecurityGroup(sgName, nodeID string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "setting security group",
			"group", sgName,
			"node_id", nodeID,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.SetSecurityGroup(sgName, nodeID)
}

func (c LoggingClient) EnsureKubernikusRulesInSecurityGroup(k *v1.Kluster) (created bool, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "ensured securitygroup",
			"group", k.Spec.Openstack.SecurityGroupName,
			"created", created,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.EnsureKubernikusRulesInSecurityGroup(k)
}

func (c LoggingClient) DeleteServerGroup(name string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "deleted servergroup",
			"name", name,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.DeleteServerGroup(name)
}

func (c LoggingClient) EnsureServerGroup(name string) (id string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "ensure servergroup",
			"name", name,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())

	return c.Client.EnsureServerGroup(name)
}

func (c LoggingClient) EnsureNodeTags(node Node, klusterName, poolName string) (added []string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "ensured node tags",
			"nodeID", node.ID,
			"tags", strings.Join(added, ","),
			"took", time.Since(begin),
			"v", 4,
			"err", err,
		)
	}(time.Now())

	return c.Client.EnsureNodeTags(node, klusterName, poolName)
}

func (c LoggingClient) EnsureMetadata(node Node, klusterName, poolName string) (ret map[string]string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "ensured node metadata",
			"nodeID", node.ID,
			"return", fmt.Sprintf("%v", ret),
			"took", time.Since(begin),
			"v", 4,
			"err", err,
		)
	}(time.Now())

	return c.Client.EnsureMetadata(node, klusterName, poolName)
}
