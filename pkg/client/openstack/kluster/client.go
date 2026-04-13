package kluster

import (
	"errors"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"
	flavorutil "github.com/gophercloud/utils/openstack/compute/v2/flavors"
	imageutil "github.com/gophercloud/utils/openstack/imageservice/v2/images"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/compute"
	"github.com/sapcc/kubernikus/pkg/templates"
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
	EnsureNodeTags(node Node, klusterName, poolName string) ([]string, error)
	EnsureMetadata(node Node, klusterName, poolName string) (map[string]string, error)
}

type klusterClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
	ImageClient    *gophercloud.ServiceClient
}

func NewKlusterClient(network, compute, identity, image *gophercloud.ServiceClient) KlusterClient {
	return &klusterClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
		ImageClient:    image,
	}
}

func (c *klusterClient) CreateNode(kluster *v1.Kluster, pool *models.NodePool, name string, userData []byte) (string, error) {
	configDrive := true

	networks := []servers.Network{{UUID: kluster.Spec.Openstack.NetworkID}}
	flavorID, err := flavorutil.IDFromName(c.ComputeClient, pool.Flavor)
	if err != nil {
		return "", fmt.Errorf("failed to find id for flavor %s: %w", pool.Flavor, err)
	}
	imageID, err := imageutil.IDFromName(c.ImageClient, pool.Image)
	if err != nil {
		return "", fmt.Errorf("failed to find id for image %s: %w", pool.Image, err)
	}

	tags := nodeTags(kluster.Spec.Name, pool.Name)
	tags = append(tags, "kubernikus:template-version="+templates.TEMPLATE_VERSION)
	tags = append(tags, "kubernikus:api-version="+kluster.Spec.Version)
	metadata := nodeMetadata(kluster.Spec.Name, pool.Name)
	metadata["kubernikus:template-version"] = templates.TEMPLATE_VERSION
	metadata["kubernikus:api-version"] = kluster.Spec.Version

	var createOpts servers.CreateOptsBuilder = servers.CreateOpts{
		Name:             name,
		FlavorRef:        flavorID,
		ImageRef:         imageID,
		AvailabilityZone: pool.AvailabilityZone,
		Networks:         networks,
		UserData:         userData,
		SecurityGroups:   []string{kluster.Spec.Openstack.SecurityGroupName},
		ConfigDrive:      &configDrive,
		Metadata:         metadata,
		Tags:             tags,
	}

	if os.Getenv("NODEPOOL_AFFINITY") != "" || os.Getenv("NODEPOOL_ANTI_AFFINITY") != "" {
		serverGroupID, err := c.EnsureServerGroup(kluster.Name + "/" + pool.Name)
		if err != nil {
			return "", fmt.Errorf("failed to ensure server group: %w", err)
		}

		createOpts = schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints:    schedulerhints.SchedulerHints{Group: serverGroupID},
		}
	}

	var server *servers.Server

	if pool.CustomRootDiskSize > 0 {
		blockDevices := []bootfromvolume.BlockDevice{{
			UUID:                imageID,
			VolumeSize:          int(pool.CustomRootDiskSize),
			BootIndex:           0,
			DeleteOnTermination: true,
			SourceType:          "image",
			DestinationType:     "volume",
		}}
		createOpts = &bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDevices,
		}

		server, err = bootfromvolume.Create(c.ComputeClient, createOpts).Extract()
	} else {
		server, err = compute.Create(c.ComputeClient, createOpts).Extract()
	}

	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
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
	var filteredNodes []Node

	err := servers.List(c.ComputeClient, servers.ListOpts{Name: "^(kks-)?" + k.Spec.Name + "-" + pool.Name + "-"}).EachPage(func(page pagination.Page) (bool, error) {
		if page != nil {
			unfilteredNodes, err := ExtractServers(page)
			if err != nil {
				return false, err
			}
			for _, node := range unfilteredNodes {
				if util.IsKubernikusNode(node.Name, k.Spec.Name, pool.Name) {
					filteredNodes = append(filteredNodes, node)
				}
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
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
		return false, fmt.Errorf("security group %v not found: %s", sgName, err)
	}

	if kluster.ClusterCIDR() == "" {
		return false, errors.New("cluster CIDR for kluster not set")
	}

	groups, err := securitygroups.ExtractGroups(page)
	if err != nil {
		return false, err
	}

	if len(groups) != 1 {
		return false, fmt.Errorf("more than one SecurityGroup with name %v found", sgName)
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
	policy := "soft-affinity"
	if os.Getenv("NODEPOOL_ANTI_AFFINITY") != "" {
		policy = "soft-anti-affinity"
	}
	sg, err := c.serverGroupByName(name)
	if err != nil {
		return "", err
	}
	if sg != nil {
		return sg.ID, nil
	}
	sg, err = servergroups.Create(c.ComputeClient, servergroups.CreateOpts{
		Name:     name,
		Policies: []string{policy},
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
	page, err := servergroups.List(c.ComputeClient, servergroups.ListOpts{}).AllPages()
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

func (c *klusterClient) EnsureNodeTags(node Node, klusterName, poolName string) ([]string, error) {

	exitingTags := sets.NewString()
	if node.Tags != nil {
		exitingTags.Insert(*node.Tags...)
	}
	missingTags := sets.NewString(nodeTags(klusterName, poolName)...).Difference(exitingTags).UnsortedList()

	added := []string{}
	for _, tag := range missingTags {
		if err := tags.Add(c.ComputeClient, node.ID, tag).ExtractErr(); err != nil {
			return added, fmt.Errorf("failed to add tag %s to instance %s, %w", tag, node.ID, err)

		}
		added = append(added, tag)
	}
	return added, nil

}

func (c *klusterClient) EnsureMetadata(node Node, klusterName, poolName string) (map[string]string, error) {

	metadata := nodeMetadata(klusterName, poolName)
	if node.Metadata == nil {
		node.Metadata = map[string]string{}
	}
	//remove metadata keys that are aleady present
	for k, v := range metadata {
		if node.Metadata[k] == v {
			delete(metadata, k)
		}
	}
	if len(metadata) == 0 {
		return nil, nil // nothing left to set
	}
	return servers.UpdateMetadata(c.ComputeClient, node.ID, servers.MetadataOpts(metadata)).Extract()
}

func nodeTags(kluster, pool string) []string {
	return []string{
		"kubernikus",
		"kubernikus:kluster=" + kluster,
		"kubernikus:nodepool=" + pool,
	}
}

func nodeMetadata(kluster, pool string) map[string]string {
	return map[string]string{
		"provisioner":         "kubernikus",
		"kubernikus:nodepool": pool,
		"kubernikus:kluster":  kluster,
	}
}
