package kluster

import (
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"
	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/compute"
)

type KlusterClient interface {
	CreateNode(*models.NodePool, string, []byte) (string, error)
	DeleteNode(string) error
	ListNodes(*models.NodePool) ([]Node, error)
	SetSecurityGroup(nodeID string) error
	EnsureKubernikusRuleInSecurityGroup() (bool, error)
}

type klusterClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient

	Kluster   *v1.Kluster
	NodeStore cache.Store
}

type cachedNodesEntry struct {
	Kluster *v1.Kluster
	Pool    *models.NodePool
	Nodes   []Node
}

func CachedNodesKeyFunc(obj interface{}) (string, error) {
	entry, ok := obj.(cachedNodesEntry)
	if !ok {
		return "", fmt.Errorf("unexpected object in cache")
	}
	return fmt.Sprintf("%v-%v", entry.Kluster.Spec.Name, entry.Pool.Name), nil
}

func NewKlusterClient(network, compute, identity *gophercloud.ServiceClient, kluster *v1.Kluster) KlusterClient {
	var client KlusterClient
	client = &klusterClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
		Kluster:        kluster,
		NodeStore:      cache.NewTTLStore(CachedNodesKeyFunc, 1*time.Minute),
	}

	return client
}

func (c *klusterClient) CreateNode(pool *models.NodePool, name string, userData []byte) (string, error) {
	configDrive := true
	server, err := compute.Create(c.ComputeClient, servers.CreateOpts{
		Name:           name,
		FlavorName:     pool.Flavor,
		ImageName:      pool.Image,
		Networks:       []servers.Network{servers.Network{UUID: c.Kluster.Spec.Openstack.NetworkID}},
		UserData:       userData,
		ServiceClient:  c.ComputeClient,
		SecurityGroups: []string{c.Kluster.Spec.Openstack.SecurityGroupName},
		ConfigDrive:    &configDrive,
	}).Extract()

	if err != nil {
		return "", err
	}

	return server.ID, nil
}

func (c *klusterClient) DeleteNode(id string) (err error) {
	err = servers.Delete(c.ComputeClient, id).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func (c *klusterClient) ListNodes(pool *models.NodePool) (nodes []Node, err error) {
	//obj, exists, err := c.NodeStore.Get(cachedNodesEntry{c.Kluster, pool, nil})
	//if err != nil {
	//  return nil, err
	//}
	//if exists {
	//  return obj.(cachedNodesEntry).Nodes, nil
	//}

	prefix := fmt.Sprintf("%v-%v-", c.Kluster.Spec.Name, pool.Name)
	err = servers.List(c.ComputeClient, servers.ListOpts{Name: prefix}).EachPage(func(page pagination.Page) (bool, error) {
		nodes, err = ExtractServers(page)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	//err = c.NodeStore.Add(cachedNodesEntry{c.Kluster, pool, nodes})
	//if err != nil {
	//  return nil, err
	//}

	return nodes, nil
}

func (c *klusterClient) SetSecurityGroup(nodeID string) (err error) {
	return secgroups.AddServer(c.ComputeClient, nodeID, c.Kluster.Spec.Openstack.SecurityGroupName).ExtractErr()
}

func (c *klusterClient) EnsureKubernikusRuleInSecurityGroup() (created bool, err error) {
	page, err := securitygroups.List(c.NetworkClient, securitygroups.ListOpts{Name: c.Kluster.Spec.Openstack.SecurityGroupName}).AllPages()
	if err != nil {
		return false, fmt.Errorf("SecurityGroup %v not found: %s", c.Kluster.Spec.Openstack.SecurityGroupName, err)
	}

	groups, err := securitygroups.ExtractGroups(page)
	if err != nil {
		return false, err
	}

	if len(groups) != 1 {
		return false, fmt.Errorf("More than one SecurityGroup with name %v found", c.Kluster.Spec.Openstack.SecurityGroupName)
	}

	udp := false
	tcp := false
	icmp := false
	for _, rule := range groups[0].Rules {
		if rule.Direction != string(rules.DirIngress) {
			continue
		}

		if rule.EtherType != string(rules.EtherType4) {
			continue
		}

		if rule.RemoteIPPrefix != c.Kluster.Spec.ClusterCIDR {
			continue
		}

		if rule.Protocol == string(rules.ProtocolICMP) {
			icmp = true
			continue
		}

		if rule.Protocol == string(rules.ProtocolUDP) {
			udp = true
			continue
		}

		if rule.Protocol == string(rules.ProtocolTCP) {
			tcp = true
			continue
		}

		if icmp && udp && tcp {
			break
		}
	}

	opts := rules.CreateOpts{
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		SecGroupID:     groups[0].ID,
		RemoteIPPrefix: c.Kluster.Spec.ClusterCIDR,
	}

	if !udp {
		opts.Protocol = rules.ProtocolUDP
		_, err := rules.Create(c.NetworkClient, opts).Extract()
		if err != nil {
			return false, err
		}
	}

	if !tcp {
		opts.Protocol = rules.ProtocolTCP
		_, err := rules.Create(c.NetworkClient, opts).Extract()
		if err != nil {
			return false, err
		}
	}

	if !icmp {
		opts.Protocol = rules.ProtocolICMP
		_, err := rules.Create(c.NetworkClient, opts).Extract()
		if err != nil {
			return false, err
		}
	}

	return !udp || !tcp || !icmp, nil
}

func ExtractServers(r pagination.Page) ([]Node, error) {
	var s []Node
	err := servers.ExtractServersInto(r, &s)
	return s, err
}
