package kluster

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/compute"
	"github.com/sapcc/kubernikus/pkg/util"
)

type KlusterClient interface {
	CreateNode(*v1.Kluster, *models.NodePool, string, []byte) (string, error)
	DeleteNode(string) error
	RebootNode(string) error
	ListNodes(*v1.Kluster, *models.NodePool) ([]Node, error)
	SetSecurityGroup(sgName, nodeID string) error
	EnsureKubernikusRuleInSecurityGroup(*v1.Kluster) (bool, error)
	EnsureServerGroup(name string) (string, error)
	DeleteServerGroup(name string) error
	CheckNodeTag(nodeID, tag string) (bool, error)
	AddNodeTag(nodeID, tag string) error
}

type klusterClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
}

func NewKlusterClient(network, compute, identity *gophercloud.ServiceClient) KlusterClient {
	var client KlusterClient
	client = &klusterClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
	}

	return client
}

func (c *klusterClient) CreateNode(kluster *v1.Kluster, pool *models.NodePool, name string, userData []byte) (string, error) {
	configDrive := true

	networks := []servers.Network{{UUID: kluster.Spec.Openstack.NetworkID}}

	if strings.HasPrefix(pool.Flavor, "zh") {
		networks = []servers.Network{
			{UUID: kluster.Spec.Openstack.NetworkID},
			{UUID: kluster.Spec.Openstack.NetworkID},
			{UUID: kluster.Spec.Openstack.NetworkID},
			{UUID: kluster.Spec.Openstack.NetworkID},
		}
	}

	if strings.HasPrefix(pool.Flavor, "zg") {
		networks = []servers.Network{
			{UUID: kluster.Spec.Openstack.NetworkID},
			{UUID: kluster.Spec.Openstack.NetworkID},
		}
	}

	var createOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             name,
		FlavorName:       pool.Flavor,
		ImageName:        pool.Image,
		AvailabilityZone: pool.AvailabilityZone,
		Networks:         networks,
		UserData:         userData,
		ServiceClient:    c.ComputeClient,
		SecurityGroups:   []string{kluster.Spec.Openstack.SecurityGroupName},
		ConfigDrive:      &configDrive,
		Metadata: map[string]string{"provisioner": "kubernikus", "nodepool": pool.Name, "kluster": kluster.Name},
	}

	if os.Getenv("NODEPOOL_AFFINITY") != "" {
		serverGroupID, err := c.EnsureServerGroup(kluster.Name + "/" + pool.Name)
		if err != nil {
			return "", err
		}

		createOpts = schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints:    schedulerhints.SchedulerHints{Group: serverGroupID},
		}
	}

	server, err := compute.Create(c.ComputeClient, createOpts).Extract()
	if err != nil {
		return "", err
	}

	return server.ID, nil
}

func (c *klusterClient) DeleteNode(id string) error {

	return servers.Delete(c.ComputeClient, id).ExtractErr()
}

func (c *klusterClient) RebootNode(id string) error {
	return servers.Reboot(c.ComputeClient, id, &servers.RebootOpts{Type: servers.SoftReboot}).ExtractErr()
}

func (c *klusterClient) ListNodes(k *v1.Kluster, pool *models.NodePool) ([]Node, error) {
	var unfilteredNodes []Node
	var filteredNodes []Node
	var err error

	err = servers.List(c.ComputeClient, servers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		if page != nil {
			unfilteredNodes, err = ExtractServers(page)
			if err != nil {
				return false, err
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	//filter nodeList https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	filteredNodes = unfilteredNodes[:0]
	for _, node := range unfilteredNodes {
		if util.IsKubernikusNode(node.Name, k.Spec.Name, pool.Name) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, nil
}

func (c *klusterClient) SetSecurityGroup(sgName, nodeID string) (err error) {
	return secgroups.AddServer(c.ComputeClient, nodeID, sgName).ExtractErr()
}

func (c *klusterClient) EnsureKubernikusRuleInSecurityGroup(kluster *v1.Kluster) (created bool, err error) {
	sgName := kluster.Spec.Openstack.SecurityGroupName
	page, err := securitygroups.List(c.NetworkClient, securitygroups.ListOpts{Name: sgName}).AllPages()
	if err != nil {
		return false, fmt.Errorf("SecurityGroup %v not found: %s", sgName, err)
	}

	if kluster.ClusterCIDR() == "" {
		return false, errors.New("Cluster CIDR for kluster not set")
	}

	groups, err := securitygroups.ExtractGroups(page)
	if err != nil {
		return false, err
	}

	if len(groups) != 1 {
		return false, fmt.Errorf("More than one SecurityGroup with name %v found", sgName)
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

		if rule.RemoteIPPrefix != *kluster.Spec.ClusterCIDR {
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
		RemoteIPPrefix: *kluster.Spec.ClusterCIDR,
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

func (c *klusterClient) EnsureServerGroup(name string) (id string, err error) {
	sg, err := c.serverGroupByName(name)
	if err != nil {
		return "", err
	}
	if sg != nil {
		return sg.ID, nil
	}
	sg, err = servergroups.Create(c.ComputeClient, servergroups.CreateOpts{
		Name:     name,
		Policies: []string{"soft-affinity"},
	}).Extract()
	if err != nil {
		return "", err
	}
	return sg.ID, nil
}

func (c *klusterClient) DeleteServerGroup(name string) error {
	sg, err := c.serverGroupByName(name)
	if err != nil {
		return err
	}
	if sg != nil {
		return servergroups.Delete(c.ComputeClient, sg.ID).ExtractErr()
	}
	return nil
}

func (c *klusterClient) serverGroupByName(name string) (*servergroups.ServerGroup, error) {
	page, err := servergroups.List(c.ComputeClient).AllPages()
	if err != nil {
		return nil, err
	}
	sgs, err := servergroups.ExtractServerGroups(page)
	if err != nil {
		return nil, err
	}
	for _, sg := range sgs {
		if sg.Name == name {
			return &sg, nil
		}
	}
	return nil, nil
}

func ExtractServers(r pagination.Page) ([]Node, error) {
	var s []Node
	err := servers.ExtractServersInto(r, &s)
	return s, err
}

func (c *klusterClient) CheckNodeTag(nodeID, tag string) (bool, error) {
	return tags.Check(c.ComputeClient, nodeID, tag).Extract()
}

func (c *klusterClient) AddNodeTag(nodeID, tag string) error {
	return tags.Add(c.ComputeClient, nodeID, tag).ExtractErr()
}
