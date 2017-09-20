package openstack

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/domains"
)

type client struct {
	klusterClients      sync.Map
	adminProviderClient *gophercloud.ProviderClient

	authURL           string
	authUsername      string
	authPassword      string
	authDomain        string
	authProject       string
	authProjectDomain string

	secrets typedv1.SecretInterface
}

type Client interface {
	CreateNode(*kubernikus_v1.Kluster, *kubernikus_v1.NodePool, []byte) (string, error)
	DeleteNode(*kubernikus_v1.Kluster, string) error
	GetNodes(*kubernikus_v1.Kluster, *kubernikus_v1.NodePool) ([]Node, error)

	GetProject(id string) (*Project, error)
	GetRegion() (string, error)
	GetRouters(project_id string) ([]Router, error)
	DeleteUser(username, domainID string) error

	CreateWormhole(*kubernikus_v1.Kluster, string, string) (string, error)
	GetWormhole(*kubernikus_v1.Kluster) (*Node, error)
}

type Project struct {
	ID       string
	Name     string
	Domain   string
	DomainID string
}

type Router struct {
	ID       string
	Networks []Network
	Subnets  []Subnet
}

type Network struct {
	ID string
}

type Subnet struct {
	ID   string
	CIDR string
}

type Node struct {
	servers.Server
	StateExt
}

func (n *Node) Starting() bool {
	// https://github.com/openstack/nova/blob/be3a66781f7fd58e5c5c0fe89b33f8098cfb0f0d/nova/objects/fields.py#L884
	if n.TaskState == "spawning" || n.TaskState == "scheduling" || n.TaskState == "networking" || n.TaskState == "block_device_mapping" {
		return true
	}

	if n.TaskState != "" {
		return false
	}

	if n.VMState == "building" {
		return true
	}

	return false
}

func (n *Node) Stopping() bool {
	if n.TaskState == "spawning" || n.TaskState == "scheduling" || n.TaskState == "networking" || n.TaskState == "block_device_mapping" {
		return false
	}

	if n.TaskState != "" {
		return true
	}

	return false
}

func (n *Node) Running() bool {
	if n.Starting() || n.Stopping() {
		return false
	}

	// 0: NOSTATE
	// 1: RUNNING
	// 3: PAUSED
	// 4: SHUTDOWN
	// 6: CRASHED
	// 7: SUSPENDED
	if n.PowerState > 1 {
		return false
	}

	//ACTIVE = 'active'
	//BUILDING = 'building'
	//PAUSED = 'paused'
	//SUSPENDED = 'suspended'
	//STOPPED = 'stopped'
	//RESCUED = 'rescued'
	//RESIZED = 'resized'
	//SOFT_DELETED = 'soft-delete'
	//DELETED = 'deleted'
	//ERROR = 'error'
	//SHELVED = 'shelved'
	//SHELVED_OFFLOADED = 'shelved_offloaded'
	if n.VMState == "active" {
		return true
	}

	return false
}

type StateExt struct {
	TaskState  string `json:"OS-EXT-STS:task_state"`
	VMState    string `json:"OS-EXT-STS:vm_state"`
	PowerState int    `json:"OS-EXT-STS:power_state"`
}

func (r *StateExt) UnmarshalJSON(b []byte) error {
	return nil
}

func NewClient(secrets typedv1.SecretInterface, klusterEvents cache.SharedIndexInformer, authURL, username, password, domain, project, projectDomain string) Client {

	c := &client{
		authURL:           authURL,
		authUsername:      username,
		authPassword:      password,
		authDomain:        domain,
		authProject:       project,
		authProjectDomain: projectDomain,
		secrets:           secrets,
	}

	klusterEvents.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if kluster, ok := obj.(*kubernikus_v1.Kluster); ok {
				glog.V(5).Info("Deleting shared openstack client for kluster %s", kluster.Name)
				c.klusterClients.Delete(kluster.GetUID())
			}
		},
	})
	return c
}

func (c *client) adminClient() (*gophercloud.ProviderClient, error) {
	if c.adminProviderClient != nil {
		return c.adminProviderClient, nil
	}

	provider, err := openstack.NewClient(c.authURL)
	if err != nil {
		return nil, err
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: c.authURL,
		Username:         c.authUsername,
		Password:         c.authPassword,
		DomainName:       c.authDomain,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: c.authProject,
			DomainName:  c.authProjectDomain,
		},
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c.adminProviderClient = provider

	return c.adminProviderClient, nil
}

func (c *client) controlPlaneClient() (*gophercloud.ProviderClient, error) {
	if c.adminProviderClient != nil {
		return c.adminProviderClient, nil
	}

	provider, err := openstack.NewClient(c.authURL)
	if err != nil {
		return nil, err
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: c.authURL,
		Username:         c.authUsername,
		Password:         c.authPassword,
		DomainName:       c.authDomain,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectID: "06a832fedd4b422bbf2d6d52be59a93d",
		},
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c.adminProviderClient = provider

	return c.adminProviderClient, nil
}

func (c *client) klusterClientFor(kluster *kubernikus_v1.Kluster) (*gophercloud.ProviderClient, error) {
	secret_name := kluster.Name

	if obj, found := c.klusterClients.Load(kluster.GetUID()); found {
		return obj.(*gophercloud.ProviderClient), nil
	}

	secret, err := c.secrets.Get(secret_name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve secret %s/%s: %v", kluster.GetNamespace(), secret_name, err)
	}

	provider, err := openstack.NewClient(string(secret.Data["openstack-auth-url"]))
	if err != nil {
		return nil, err
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: string(secret.Data["openstack-auth-url"]),
		Username:         string(secret.Data["openstack-username"]),
		Password:         string(secret.Data["openstack-password"]),
		DomainName:       string(secret.Data["openstack-domain-name"]),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectID: kluster.Spec.OpenstackInfo.ProjectID,
		},
	}

	glog.V(5).Infof("AuthOptions: %#v", authOptions)

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c.klusterClients.Store(kluster.GetUID(), provider)

	return provider, nil
}

func (c *client) GetProject(id string) (*Project, error) {
	provider, err := c.adminClient()
	if err != nil {
		return nil, err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	project, err := projects.Get(identity, id).Extract()
	if err != nil {
		return nil, err
	}

	domain, err := domains.Get(identity, project.DomainID).Extract()
	if err != nil {
		return nil, err
	}

	return &Project{ID: id, Name: project.Name, DomainID: project.DomainID, Domain: domain.Name}, nil
}

func (c *client) GetRouters(project_id string) ([]Router, error) {
	provider, err := c.adminClient()
	if err != nil {
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	resultList := []Router{}
	err = routers.List(networkClient, routers.ListOpts{TenantID: project_id}).EachPage(func(page pagination.Page) (bool, error) {
		routerList, err := routers.ExtractRouters(page)
		if err != nil {
			return false, err
		}
		for _, router := range routerList {
			resultRouter := Router{ID: router.ID, Subnets: []Subnet{}}
			networkIDs, err := getRouterNetworks(networkClient, router.ID)
			if err != nil {
				return false, err
			}
			for _, networkID := range networkIDs {
				network, err := networks.Get(networkClient, networkID).Extract()
				if err != nil {
					return false, err
				}
				resultRouter.Networks = append(resultRouter.Networks, Network{ID: network.ID})

				for _, subnetID := range network.Subnets {
					subnet, err := subnets.Get(networkClient, subnetID).Extract()
					if err != nil {
						return false, err
					}
					resultRouter.Subnets = append(resultRouter.Subnets, Subnet{ID: subnet.ID, CIDR: subnet.CIDR})
				}
			}
			if len(resultRouter.Subnets) > 0 {
				resultList = append(resultList, resultRouter)
			}
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}
	return resultList, nil

}

func (c *client) DeleteUser(username, domainID string) error {
	provider, err := c.adminClient()
	if err != nil {
		return err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}
	return users.List(identity, users.ListOpts{DomainID: domainID, Name: username}).EachPage(func(page pagination.Page) (bool, error) {
		userList, err := users.ExtractUsers(page)
		if err != nil {
			return false, err
		}
		switch len(userList) {
		case 0:
			return false, nil
		case 1:
			return false, users.Delete(identity, userList[0].ID).ExtractErr()
		default:
			return false, errors.New("Multiple users found")
		}
	})
}

func getRouterNetworks(client *gophercloud.ServiceClient, routerID string) ([]string, error) {
	networks := []string{}
	err := ports.List(client, ports.ListOpts{DeviceID: routerID, DeviceOwner: "network:router_interface"}).EachPage(func(page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, port := range portList {
			networks = append(networks, port.NetworkID)
		}
		return true, nil
	})
	return networks, err
}

func (c *client) GetNodes(kluster *kubernikus_v1.Kluster, pool *kubernikus_v1.NodePool) ([]Node, error) {
	project_id := kluster.Spec.OpenstackInfo.RouterID
	pool_id := pool.Name

	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return nil, err
	}

	nodes := []Node{}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nodes, err
	}
	glog.V(5).Infof("Listing nodes for %v/%v", project_id, pool_id)

	prefix := fmt.Sprintf("kubernikus-%v-%v", kluster.Spec.Name, pool_id)
	opts := servers.ListOpts{Name: prefix}

	servers.List(client, opts).EachPage(func(page pagination.Page) (bool, error) {
		nodes, err = ExtractServers(page)
		if err != nil {
			glog.V(5).Infof("Couldn't extract server %v", err)
			return false, err
		}

		return true, nil
	})

	return nodes, nil
}

func (c *client) GetWormhole(kluster *kubernikus_v1.Kluster) (*Node, error) {
	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("wormhole-%v", kluster.Name)
	opts := servers.ListOpts{Name: prefix}

	var node *Node
	servers.List(client, opts).EachPage(func(page pagination.Page) (bool, error) {
		serverList, err := ExtractServers(page)
		if err != nil {
			glog.V(5).Infof("Couldn't extract server %v", err)
			return false, err
		}

		if len(serverList) > 0 {
			node = &serverList[0]
		}

		return true, nil
	})

	return node, nil
}

func (c *client) CreateNode(kluster *kubernikus_v1.Kluster, pool *kubernikus_v1.NodePool, userData []byte) (string, error) {
	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return "", err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	name := v1.SimpleNameGenerator.GenerateName(fmt.Sprintf("kubernikus-%v-%v-", kluster.Spec.Name, pool.Name))
	glog.V(5).Infof("Creating node %v", name)

	server, err := servers.Create(client, servers.CreateOpts{
		Name:          name,
		FlavorName:    pool.Flavor,
		ImageName:     pool.Image,
		Networks:      []servers.Network{servers.Network{UUID: kluster.Spec.OpenstackInfo.NetworkID}},
		UserData:      userData,
		ServiceClient: client,
	}).Extract()

	if err != nil {
		glog.V(5).Infof("Couldn't create node %v: %v", name, err)
		return "", err
	}

	return server.ID, nil
}

func (c *client) CreateWormhole(kluster *kubernikus_v1.Kluster, projectID, networkID string) (string, error) {
	provider, err := c.controlPlaneClient()
	if err != nil {
		return "", fmt.Errorf("Couldn't get provider for %v: %v", projectID, err)
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("Couldn't get Compute client: %v", err)
	}

	name := fmt.Sprintf("wormhole-%v", kluster.Name)
	glog.V(5).Infof("Creating %v", name)

	localPort, err := c.FindOrCreateWormholeLocalPort(kluster, projectID, networkID)
	if err != nil {
		return "", fmt.Errorf("Couldn't find/create local wormhole: %v", err)
	}

	foreignPort, err := c.FindOrCreateWormholeForeignPort(kluster, networkID)
	if err != nil {
		return "", fmt.Errorf("Couldn't find/create local wormhol: %v", err)
	}

	glog.Infof("%#v", servers.CreateOpts{
		Name:          name,
		FlavorName:    "m1.small",
		ImageName:     "ubuntu-16.04-amd64-vmware",
		Networks:      []servers.Network{servers.Network{Port: foreignPort}, servers.Network{Port: localPort}},
		ServiceClient: client,
	})

	server, err := servers.Create(client, servers.CreateOpts{
		Name:          name,
		FlavorName:    "m1.small",
		ImageName:     "ubuntu-16.04-amd64-vmware",
		Networks:      []servers.Network{servers.Network{Port: foreignPort}, servers.Network{Port: localPort}},
		ServiceClient: client,
	}).Extract()

	if err != nil {
		glog.V(5).Infof("Couldn't create %v: %v", name, err)
		return "", fmt.Errorf("Couldn't create wormhole: %v", err)
	}

	return server.ID, nil
}

func (c *client) DeleteNode(kluster *kubernikus_v1.Kluster, ID string) error {
	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	err = servers.Delete(client, ID).ExtractErr()
	if err != nil {
		glog.V(5).Infof("Couldn't delete node %v: %v", kluster.Name, err)
		return err
	}

	return nil
}

func (c *client) GetRegion() (string, error) {
	provider, err := c.adminClient()
	if err != nil {
		return "", err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	opts := services.ListOpts{ServiceType: "compute"}
	computeServiceID := ""
	err = services.List(identity, opts).EachPage(func(page pagination.Page) (bool, error) {
		serviceList, err := services.ExtractServices(page)
		if err != nil {
			return false, err
		}

		if computeServiceID == "" {
			computeServiceID = serviceList[0].ID
		}

		return true, nil
	})

	if err != nil {
		return "", err
	}

	if computeServiceID == "" {
		return "", fmt.Errorf("Couldn't find a compute service. Bailing out.")
	}

	endpointOpts := endpoints.ListOpts{Availability: gophercloud.AvailabilityPublic, ServiceID: computeServiceID}
	region := ""
	err = endpoints.List(identity, endpointOpts).EachPage(func(page pagination.Page) (bool, error) {
		endpoints, err := endpoints.ExtractEndpoints(page)
		if err != nil {
			return false, err
		}

		if region == "" {
			region = endpoints[0].Region
		}

		return true, nil
	})

	if err != nil {
		return "", err
	}

	if region == "" {
		return "", fmt.Errorf("Couldn't find the region. Bailing out.")
	}

	return region, nil
}

func (c *client) GetWormholeForeignPort(kluster *kubernikus_v1.Kluster) (string, error) {
	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return "", fmt.Errorf("Couldn't create foreign wormhole port: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("Couldn't create foreign wormhole port: %v", err)
	}

	name := fmt.Sprintf("kubernikus:wormhole-foreign-%v", kluster.Name)
	id, err := ports.IDFromName(client, name)
	if err != nil {
		return "", err
	}

	port, err := ports.Get(client, id).Extract()
	if err != nil {
		return "", err
	}

	return port.ID, nil
}

func (c *client) GetWormholeLocalPort(kluster *kubernikus_v1.Kluster) (string, error) {
	provider, err := c.controlPlaneClient()
	if err != nil {
		return "", fmt.Errorf("Couldn't create local wormhole port: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("Couldn't create local wormhole port: %v", err)
	}

	name := fmt.Sprintf("kubernikus:wormhole-local-%v", kluster.Name)
	id, err := ports.IDFromName(client, name)
	if err != nil {
		return "", err
	}

	port, err := ports.Get(client, id).Extract()
	if err != nil {
		return "", err
	}

	return port.ID, nil
}

func (c *client) CreateWormholeLocalPort(kluster *kubernikus_v1.Kluster, projectID, networkID string) (string, error) {
	provider, err := c.controlPlaneClient()
	if err != nil {
		return "", fmt.Errorf("Couldn't create local wormhole port: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("Couldn't create local wormhole port: %v", err)
	}

	name := fmt.Sprintf("kubernikus:wormhole-local-%v", kluster.Name)
	port, err := ports.Create(client, ports.CreateOpts{
		Name:      name,
		NetworkID: networkID,
	}).Extract()

	if err != nil {
		return "", fmt.Errorf("Couldn't create local wormhole port: %v", err)
	}

	return port.ID, nil
}

func (c *client) CreateWormholeForeignPort(kluster *kubernikus_v1.Kluster) (string, error) {
	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return "", fmt.Errorf("Couldn't create foreign wormhole port: %v", err)
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("Couldn't create foreign wormhole port: %v", err)
	}

	name := fmt.Sprintf("kubernikus:wormhole-foreign-%v", kluster.Name)
	port, err := ports.Create(client, ports.CreateOpts{
		Name:      name,
		NetworkID: kluster.Spec.OpenstackInfo.NetworkID,
	}).Extract()

	if err != nil {
		return "", fmt.Errorf("Couldn't create foreign wormhole port: %v", err)
	}

	return port.ID, nil
}

func (c *client) FindOrCreateWormholeForeignPort(kluster *kubernikus_v1.Kluster, networkID string) (string, error) {
	id, err := c.GetWormholeForeignPort(kluster)
	if err != nil {
		if _, ok := err.(gophercloud.ErrResourceNotFound); ok {
			return c.CreateWormholeForeignPort(kluster)
		} else {
			return "", fmt.Errorf("Couldn't find or create foreign wormhole port: %v", err)
		}
	}
	return id, nil
}

func (c *client) FindOrCreateWormholeLocalPort(kluster *kubernikus_v1.Kluster, projectID, networkID string) (string, error) {
	id, err := c.GetWormholeLocalPort(kluster)
	if err != nil {
		if _, ok := err.(gophercloud.ErrResourceNotFound); ok {
			return c.CreateWormholeLocalPort(kluster, projectID, networkID)
		} else {
			return "", fmt.Errorf("Couldn't find or create local wormhole port: %v", err)
		}
	}
	return id, nil
}

func ExtractServers(r pagination.Page) ([]Node, error) {
	var s []Node
	err := servers.ExtractServersInto(r, &s)
	return s, err
}
