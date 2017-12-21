package openstack

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/compute"
	"github.com/sapcc/kubernikus/pkg/client/openstack/domains"
	"github.com/sapcc/kubernikus/pkg/client/openstack/roles"
)

var serviceUserRoles = []string{"network_admin", "member"}

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

	domainNameToID sync.Map
	roleNameToID   sync.Map

	Logger log.Logger
}

type Client interface {
	CreateNode(*kubernikus_v1.Kluster, *models.NodePool, []byte) (string, error)
	DeleteNode(*kubernikus_v1.Kluster, string) error
	GetNodes(*kubernikus_v1.Kluster, *models.NodePool) ([]Node, error)

	GetProject(id string) (*Project, error)
	GetRegion() (string, error)
	GetRouters(project_id string) ([]Router, error)
	DeleteUser(username, domainID string) error
	CreateKlusterServiceUser(username, password, domain, defaultProjectID string) error
	GetKubernikusCatalogEntry() (string, error)
	GetSecurityGroupID(project_id, name string) (string, error)
}

type Project struct {
	ID       string
	Name     string
	Domain   string
	DomainID string
}

type Router struct {
	ID                string
	ExternalNetworkID string
	Networks          []Network
}

type Network struct {
	ID      string
	Subnets []Subnet
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

func NewClient(secrets typedv1.SecretInterface, klusterEvents cache.SharedIndexInformer, authURL, username, password, domain, project, projectDomain string, logger log.Logger) Client {

	c := &client{
		authURL:           authURL,
		authUsername:      username,
		authPassword:      password,
		authDomain:        domain,
		authProject:       project,
		authProjectDomain: projectDomain,
		secrets:           secrets,
		Logger:            log.With(logger, "client", "openstack"),
	}

	klusterEvents.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if kluster, ok := obj.(*kubernikus_v1.Kluster); ok {
				c.Logger.Log(
					"msg", "deleting shared openstack client",
					"kluster", kluster.Name,
					"v", 5)
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
			ProjectID: string(secret.Data["openstack-project-id"]),
		},
	}

	c.Logger.Log(
		"msg", "using authOptions from secret",
		"identity_endpoint", authOptions.IdentityEndpoint,
		"username", authOptions.Username,
		"domain_name", authOptions.DomainName,
		"project_id", authOptions.Scope.ProjectID,
		"v", 5)

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
	resultRouters := []Router{}
	err = routers.List(networkClient, routers.ListOpts{TenantID: project_id}).EachPage(func(page pagination.Page) (bool, error) {
		routers, err := routers.ExtractRouters(page)
		if err != nil {
			return false, err
		}
		for _, router := range routers {
			resultRouter := Router{ID: router.ID, ExternalNetworkID: router.GatewayInfo.NetworkID}
			networkIDs, err := getRouterNetworks(networkClient, router.ID)
			if err != nil {
				return false, err
			}
			for _, networkID := range networkIDs {
				network, err := networks.Get(networkClient, networkID).Extract()
				if err != nil {
					return false, err
				}
				resultNetwork := Network{ID: network.ID, Subnets: make([]Subnet, 0, len(network.Subnets))}

				for _, subnetID := range network.Subnets {
					subnet, err := subnets.Get(networkClient, subnetID).Extract()
					if err != nil {
						return false, err
					}
					resultNetwork.Subnets = append(resultNetwork.Subnets, Subnet{ID: subnet.ID, CIDR: subnet.CIDR})
				}
				resultRouter.Networks = append(resultRouter.Networks, resultNetwork)
			}
			resultRouters = append(resultRouters, resultRouter)
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}
	return resultRouters, nil

}

func (c *client) getRoleID(client *gophercloud.ServiceClient, roleName string) (string, error) {
	if id, ok := c.roleNameToID.Load(roleName); ok {
		return id.(string), nil
	}
	err := roles.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
		roles, err := roles.ExtractRoles(page)
		if err != nil {
			return false, err
		}
		for _, role := range roles {
			c.roleNameToID.Store(role.Name, role.ID)
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	if id, ok := c.roleNameToID.Load(roleName); ok {
		return id.(string), nil
	}

	return "", fmt.Errorf("Role %s not found", roleName)

}

func (c *client) getUserByName(client *gophercloud.ServiceClient, username, domainID string) (*users.User, error) {
	var user *users.User
	err := users.List(client, users.ListOpts{DomainID: domainID, Name: username}).EachPage(func(page pagination.Page) (bool, error) {
		users, err := users.ExtractUsers(page)
		if err != nil {
			return false, err
		}
		switch len(users) {
		case 0:
			return false, nil
		case 1:
			user = &users[0]
			return false, nil
		default:
			return false, errors.New("More then one user found")
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (c *client) CreateKlusterServiceUser(username, password, domain, projectID string) error {
	provider, err := c.adminClient()
	if err != nil {
		return err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}
	domainID, err := c.getDomainID(identity, domain)
	if err != nil {
		return err
	}

	user, err := c.getUserByName(identity, username, domainID)
	if err != nil {
		return err
	}
	//Do we need to update or create?
	if user != nil {
		user, err = users.Update(identity, user.ID, users.UpdateOpts{
			Password:         password,
			DefaultProjectID: projectID,
			Description:      "Kubernikus kluster service user",
		}).Extract()
	} else {
		user, err = users.Create(identity, users.CreateOpts{
			Name:             username,
			DomainID:         domainID,
			Password:         password,
			DefaultProjectID: projectID,
			Description:      "Kubernikus kluster service user",
		}).Extract()
	}
	if err != nil {
		return err
	}
	for _, roleName := range serviceUserRoles {
		roleID, err := c.getRoleID(identity, roleName)
		if err != nil {
			return err
		}
		err = roles.AssignToUserInProject(identity, projectID, user.ID, roleID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) DeleteUser(username, domainName string) error {
	provider, err := c.adminClient()
	if err != nil {
		return err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}
	domainID, err := c.getDomainID(identity, domainName)
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

func (c *client) getDomainID(client *gophercloud.ServiceClient, domainName string) (string, error) {
	if id, ok := c.domainNameToID.Load(domainName); ok {
		return id.(string), nil
	}
	identity, err := openstack.NewIdentityV3(c.adminProviderClient, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}
	err = domains.List(identity, &domains.ListOpts{Name: domainName}).EachPage(func(page pagination.Page) (bool, error) {
		domains, err := domains.ExtractDomains(page)
		if err != nil {
			return false, err
		}
		switch len(domains) {
		case 0:
			return false, fmt.Errorf("Domain %s not found", domainName)
		case 1:
			c.domainNameToID.Store(domainName, domains[0].ID)
			return false, nil
		default:
			return false, errors.New("More then one domain found")
		}
	})
	id, _ := c.domainNameToID.Load(domainName)
	return id.(string), nil
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

func (c *client) GetNodes(kluster *kubernikus_v1.Kluster, pool *models.NodePool) ([]Node, error) {
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

	prefix := fmt.Sprintf("%v-%v-", kluster.Spec.Name, pool_id)
	opts := servers.ListOpts{Name: prefix}

	err = servers.List(client, opts).EachPage(func(page pagination.Page) (bool, error) {
		nodes, err = ExtractServers(page)
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return nodes, err
	}

	return nodes, nil
}

func (c *client) CreateNode(kluster *kubernikus_v1.Kluster, pool *models.NodePool, userData []byte) (id string, err error) {
	var name string

	defer func() {
		c.Logger.Log(
			"msg", "created node",
			"kluster", kluster.Name,
			"project", kluster.Account(),
			"name", name,
			"id", id,
			"v", 5,
			"err", err)
	}()

	name = v1.SimpleNameGenerator.GenerateName(fmt.Sprintf("%v-%v-", kluster.Spec.Name, pool.Name))

	provider, err := c.klusterClientFor(kluster)
	if err != nil {
		return "", err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	name = v1.SimpleNameGenerator.GenerateName(fmt.Sprintf("%v-%v-", kluster.Spec.Name, pool.Name))

	server, err := servers.Create(client, compute.CreateOpts{
		CreateOpts: servers.CreateOpts{
			Name:           name,
			FlavorName:     pool.Flavor,
			ImageName:      pool.Image,
			Networks:       []servers.Network{servers.Network{UUID: kluster.Spec.Openstack.NetworkID}},
			UserData:       userData,
			ServiceClient:  client,
			SecurityGroups: []string{kluster.Spec.Openstack.SecurityGroupID},
		},
	}).Extract()

	if err != nil {
		return "", err
	}

	return server.ID, nil
}

func (c *client) DeleteNode(kluster *kubernikus_v1.Kluster, ID string) (err error) {
	defer func() {
		c.Logger.Log(
			"msg", "deleted node",
			"kluster", kluster.Name,
			"project", kluster.Account(),
			"id", ID,
			"v", 5,
			"err", err)
	}()

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

func ExtractServers(r pagination.Page) ([]Node, error) {
	var s []Node
	err := servers.ExtractServersInto(r, &s)
	return s, err
}

func (c *client) GetKubernikusCatalogEntry() (string, error) {
	provider, err := c.adminClient()
	if err != nil {
		return "", err
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	catalog, err := tokens.Get(identity, provider.TokenID).ExtractServiceCatalog()
	if err != nil {
		return "", err
	}

	for _, service := range catalog.Entries {
		if service.Type == "kubernikus" {
			for _, endpoint := range service.Endpoints {
				if endpoint.Interface == "public" {
					return endpoint.URL, nil
				}
			}
		}
	}

	return "", err
}
func (c *client) GetSecurityGroupID(project_id string, name string) (string, error) {

	provider, err := c.adminClient()
	if err != nil {
		return "", err
	}
	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	var group securitygroups.SecGroup

	err = securitygroups.List(networkClient, securitygroups.ListOpts{Name: name, TenantID: project_id}).EachPage(func(page pagination.Page) (bool, error) {
		groups, err := securitygroups.ExtractGroups(page)
		if err != nil {
			return false, err
		}
		switch len(groups) {
		case 0:
			return false, errors.New("Security group not found")
		case 1:
			group = groups[0]
			return false, nil
		default:
			return false, errors.New("Multiple security groups with the same name found")
		}
	})
	return group.ID, err
}
