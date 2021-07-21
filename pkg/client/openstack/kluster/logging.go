package kluster

import (
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

func (c LoggingClient) EnsureKubernikusRuleInSecurityGroup(k *v1.Kluster) (created bool, err error) {
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

	return c.Client.EnsureKubernikusRuleInSecurityGroup(k)
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

func (c LoggingClient) CheckNodeTag(nodeID, tag string) (t bool, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "check node tag",
			"nodeID", nodeID,
			"tag", tag,
			"check", t,
			"took", time.Since(begin),
			"v", 4,
			"err", err,
		)
	}(time.Now())

	return c.Client.CheckNodeTag(nodeID, tag)
}

func (c LoggingClient) AddNodeTag(nodeID, tag string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "add node tag",
			"nodeID", nodeID,
			"tag", tag,
			"took", time.Since(begin),
			"v", 4,
			"err", err,
		)
	}(time.Now())

	return c.Client.AddNodeTag(nodeID, tag)
}
