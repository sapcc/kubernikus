package openstack

import (
	"errors"
	"fmt"

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
	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"k8s.io/client-go/informers"
	informers_core "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/sapcc/kubernikus/pkg/client/openstack/domains"
)

type client struct {
	domainProviders  map[string]*gophercloud.ProviderClient
	projectProviders map[string]*gophercloud.ProviderClient

	authURL           string
	authUsername      string
	authPassword      string
	authDomain        string
	authProject       string
	authProjectDomain string

	secrets informers_core.SecretInformer
}

type Client interface {
	CreateNode(*kubernikus_v1.Kluster, *kubernikus_v1.NodePool) (string, error)
	GetNodes(*kubernikus_v1.Kluster, *kubernikus_v1.NodePool) ([]Node, error)
	GetProject(id string) (*Project, error)
	GetRegion() (string, error)
	GetRouters(project_id string) ([]Router, error)
	DeleteUser(username, domainID string) error
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
	ID     string
	Name   string
	Status string
}

func NewClient(informers informers.SharedInformerFactory, authURL, username, password, domain, project, projectDomain string) Client {
	informers.Core().V1().Secrets().Informer()

	return &client{
		domainProviders:   make(map[string]*gophercloud.ProviderClient),
		projectProviders:  make(map[string]*gophercloud.ProviderClient),
		authURL:           authURL,
		authUsername:      username,
		authPassword:      password,
		authDomain:        domain,
		authProject:       project,
		authProjectDomain: projectDomain,
		secrets:           informers.Core().V1().Secrets(),
	}
}

func (c *client) domainProvider() (*gophercloud.ProviderClient, error) {
	return c.domainProviderFor(c.authDomain)
}

func (c *client) domainProviderFor(domain string) (*gophercloud.ProviderClient, error) {
	if c.domainProviders[domain] != nil {
		return c.domainProviders[domain], nil
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

	c.domainProviders[domain] = provider

	return c.domainProviders[domain], nil
}

func (c *client) projectProviderFor(kluster *kubernikus_v1.Kluster) (*gophercloud.ProviderClient, error) {
	project_id := kluster.Spec.OpenstackInfo.ProjectID
	secret_name := kluster.Name

	if c.projectProviders[project_id] != nil {
		return c.projectProviders[project_id], nil
	}

	secret, err := c.secrets.Lister().Secrets("kubernikus").Get(secret_name)
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve secret kubernikus/%v: %v", secret_name, err)
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
			ProjectID: project_id,
		},
	}

	glog.V(5).Infof("AuthOptions: %#v", authOptions)

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c.projectProviders[project_id] = provider

	return c.projectProviders[project_id], nil
}

func (c *client) GetProject(id string) (*Project, error) {
	provider, err := c.domainProvider()
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
	provider, err := c.domainProvider()
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
	provider, err := c.domainProvider()
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

	provider, err := c.projectProviderFor(kluster)
	if err != nil {
		return nil, err
	}

	nodes := []Node{}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nodes, err
	}
	glog.V(5).Infof("Listing nodes for %v/%v", project_id, pool_id)

	prefix := fmt.Sprintf("kubernikus-%v", pool_id)
	opts := servers.ListOpts{Name: prefix}

	servers.List(client, opts).EachPage(func(page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			glog.V(5).Infof("Couldn't extract server %v", err)
			return false, err
		}

		for _, s := range serverList {
			glog.V(5).Infof("Found node %v", s.ID)
			nodes = append(nodes, Node{ID: s.ID, Name: s.Name, Status: s.Status})
		}

		return true, nil
	})

	return nodes, nil
}

func (c *client) CreateNode(kluster *kubernikus_v1.Kluster, pool *kubernikus_v1.NodePool) (string, error) {
	provider, err := c.projectProviderFor(kluster)
	if err != nil {
		return "", err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", err
	}

	name := v1.SimpleNameGenerator.GenerateName(fmt.Sprintf("kubernikus-%v-", pool.Name))
	glog.V(5).Infof("Creating node %v", name)

	server, err := servers.Create(client, servers.CreateOpts{
		Name:          name,
		FlavorName:    pool.Flavor,
		ImageName:     pool.Image,
		Networks:      []servers.Network{servers.Network{UUID: kluster.Spec.OpenstackInfo.NetworkID}},
		ServiceClient: client,
	}).Extract()

	if err != nil {
		glog.V(5).Infof("Couldn't create node %v: %v", name, err)
		return "", err
	}

	return server.ID, nil
}

func (c *client) GetRegion() (string, error) {
	provider, err := c.domainProvider()
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
