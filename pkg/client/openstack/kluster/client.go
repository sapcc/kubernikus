package kluster

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

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
	EnsureKubernikusRulesInSecurityGroup(*v1.Kluster) (bool, error)
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
	var client KlusterClient
	client = &klusterClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
		ImageClient:    image,
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
	flavorID, err := flavorutil.IDFromName(c.ComputeClient, pool.Flavor)
	if err != nil {
		return "", fmt.Errorf("Failed to find id for flavor %s: %w", pool.Flavor, err)
	}
	imageID, err := imageutil.IDFromName(c.ImageClient, pool.Image)
	if err != nil {
		return "", fmt.Errorf("Failed to find id for image %s: %w", pool.Image, err)
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

	if os.Getenv("NODEPOOL_AFFINITY") != "" {
		serverGroupID, err := c.EnsureServerGroup(kluster.Name + "/" + pool.Name)
		if err != nil {
			return "", fmt.Errorf("Failed to ensure server group: %w", err)
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
		return "", fmt.Errorf("Failed to create node: %w", err)
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

	err := servers.List(c.ComputeClient, servers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		if page != nil {
			nodes, err := ExtractServers(page)
			if err != nil {
				return false, err
			}
			unfilteredNodes = append(unfilteredNodes, nodes...)
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

func (c *klusterClient) EnsureKubernikusRulesInSecurityGroup(kluster *v1.Kluster) (created bool, err error) {
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

	apiIP, err := ipForUrl(kluster.Status.Apiserver)
	if err != nil {
		return false, fmt.Errorf("Failed to lookup apiserver ip: %w", err)
	}

	wantedRules := []rules.SecGroupRule{
		{
			Direction:      string(rules.DirIngress),
			EtherType:      string(rules.EtherType4),
			RemoteIPPrefix: *kluster.Spec.ClusterCIDR,
			Description:    fmt.Sprintf(`Kubernikus: accept traffic from pod CIDR of cluster "%s"`, kluster.Spec.Name),
		},
		{
			Direction:    string(rules.DirEgress),
			EtherType:    string(rules.EtherType4),
			Protocol:     string(rules.ProtocolUDP),
			PortRangeMin: 123,
			PortRangeMax: 123,
			Description:  "Kubernikus: allow ntp client traffic",
		},
		{
			Direction:      string(rules.DirEgress),
			EtherType:      string(rules.EtherType4),
			Protocol:       string(rules.ProtocolTCP),
			PortRangeMin:   443,
			PortRangeMax:   443,
			RemoteIPPrefix: apiIP.String(),
			Description:    fmt.Sprintf(`Kubernikus: allow access to apiserver of cluster "%s"`, kluster.Spec.Name),
		},
	}
	if osURL, err := c.ComputeClient.ProviderClient.EndpointLocator(gophercloud.EndpointOpts{Type: "object-store", Availability: gophercloud.AvailabilityPublic}); err == nil {
		if ip, err := ipForUrl(osURL); err == nil {
			wantedRules = append(wantedRules, rules.SecGroupRule{
				Direction:      string(rules.DirEgress),
				EtherType:      string(rules.EtherType4),
				Protocol:       string(rules.ProtocolTCP),
				PortRangeMin:   443,
				PortRangeMax:   443,
				RemoteIPPrefix: ip.String(),
				Description:    `Kubernikus: allow access to regional object-store/swift`,
			})
		} else {
			fmt.Println("parse error object-store", osURL, err)
		}
	} else {
		fmt.Println("no object-store", err, osURL)
	}
	if keppelURL, err := c.ComputeClient.ProviderClient.EndpointLocator(gophercloud.EndpointOpts{Type: "keppel", Availability: gophercloud.AvailabilityPublic}); err == nil {
		if ip, err := ipForUrl(keppelURL); err == nil {
			wantedRules = append(wantedRules, rules.SecGroupRule{
				Direction:      string(rules.DirEgress),
				EtherType:      string(rules.EtherType4),
				Protocol:       string(rules.ProtocolTCP),
				PortRangeMin:   443,
				PortRangeMax:   443,
				RemoteIPPrefix: ip.String(),
				Description:    `Kubernikus: allow access to regional keppel`,
			})
		}
	}
OUTER:
	for n, wanted := range wantedRules {
		for _, rule := range groups[0].Rules {
			if MatchRule(wanted, rule) {
				continue OUTER //wanted rule is already coverd by exiting rules in group
			}
		}
		//we need to create the

		opts := rules.CreateOpts{
			Direction:      rules.RuleDirection(wanted.Direction),
			EtherType:      rules.RuleEtherType(wanted.EtherType),
			Protocol:       rules.RuleProtocol(wanted.Protocol),
			SecGroupID:     groups[0].ID,
			RemoteIPPrefix: wanted.RemoteIPPrefix,
			Description:    wanted.Description,
			PortRangeMin:   wanted.PortRangeMin,
			PortRangeMax:   wanted.PortRangeMax,
		}
		_, err := rules.Create(c.NetworkClient, opts).Extract()
		if err != nil {
			return false, fmt.Errorf("Failed to create security group %v: %w", wanted, err)
		}
		created = true
		//super extra special hack, when we create the first rule we clean out the old rules we created previously
		// can be removed after it has been rolled out to all regions
		if n == 0 {
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
				// rules regarding the clusterCIDR with a non-empty protocol are deprecated
				if rule.Protocol != "" {
					if err := rules.Delete(c.NetworkClient, rule.ID).ExtractErr(); err != nil {
						return created, err
					}
				}
			}

		}
	}

	return created, nil

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
			return added, fmt.Errorf("Failed to add tag %s to instance %s, %w", tag, node.ID, err)

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

func ipForUrl(theurl string) (net.IP, error) {
	u, err := url.Parse(theurl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url %s: %w", theurl, err)
	}
	//if host is an IP we are done
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		return ip, nil
	}
	ips, err := net.LookupHost(u.Hostname())
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("Failed to resolve host: %w", err)
	}
	if ip := net.ParseIP(ips[0]); ip != nil {
		return ip, nil
	}
	return nil, fmt.Errorf("Failed to parse resolved ip %s", ips[0])
}
