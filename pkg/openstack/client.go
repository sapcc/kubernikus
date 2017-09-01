package openstack

import (
	"errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/sapcc/kubernikus/pkg/openstack/domains"
)

type client struct {
	provider *gophercloud.ProviderClient
}

type Client interface {
	GetProject(id string) (*Project, error)
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
	ID      string
	Subnets []Subnet
}

type Subnet struct {
	ID   string
	CIDR string
}

func NewClient(authURL, username, password, domain, project, projectDomain string) (Client, error) {

	provider, err := openstack.NewClient(authURL)
	if err != nil {
		return nil, err
	}
	err = openstack.AuthenticateV3(provider, &tokens.AuthOptions{
		IdentityEndpoint: authURL,
		Username:         username,
		Password:         password,
		DomainName:       domain,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: project,
			DomainName:  projectDomain,
		},
	}, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	return &client{provider: provider}, nil

}

func (c *client) GetProject(id string) (*Project, error) {
	identity, err := openstack.NewIdentityV3(c.provider, gophercloud.EndpointOpts{})
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
	networkClient, err := openstack.NewNetworkV2(c.provider, gophercloud.EndpointOpts{})
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
	identity, err := openstack.NewIdentityV3(c.provider, gophercloud.EndpointOpts{})
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
