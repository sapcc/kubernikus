package scoped

import (
	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/client/openstack/util"
	utillog "github.com/sapcc/kubernikus/pkg/util/log"
)

type client struct {
	util.AuthenticatedClient
	Logger log.Logger
}

type Client interface {
	Authenticate(*tokens.AuthOptions) error
	GetMetadata() (*models.OpenstackMetadata, error)
}

func NewClient(authOptions *tokens.AuthOptions, logger log.Logger) (Client, error) {
	logger = utillog.NewAuthLogger(logger, authOptions)

	var c Client
	c = &client{Logger: logger}
	c = LoggingClient{c, logger}

	return c, c.Authenticate(authOptions)
}

func (c *client) Authenticate(authOptions *tokens.AuthOptions) error {
	providerClient, err := utillog.NewLoggingProviderClient(authOptions.IdentityEndpoint, c.Logger)
	if err != nil {
		return err
	}

	if err := openstack.AuthenticateV3(providerClient, authOptions, gophercloud.EndpointOpts{}); err != nil {
		return err
	}

	if c.IdentityClient, err = openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{}); err != nil {
		return err
	}

	if c.ComputeClient, err = openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{}); err != nil {
		return err
	}

	if c.NetworkClient, err = openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{}); err != nil {
		return err
	}

	return nil
}

func (c *client) GetMetadata() (metadata *models.OpenstackMetadata, err error) {
	metadata = &models.OpenstackMetadata{
		Flavors:        make([]*models.Flavor, 0),
		KeyPairs:       make([]*models.KeyPair, 0),
		Routers:        make([]*models.Router, 0),
		SecurityGroups: make([]*models.SecurityGroup, 0),
	}

	if metadata.Routers, err = c.getRouters(); err != nil {
		return metadata, err
	}

	if metadata.KeyPairs, err = c.getKeyPairs(); err != nil {
		return metadata, err
	}

	if metadata.SecurityGroups, err = c.getSecurityGroups(); err != nil {
		return metadata, err
	}

	if metadata.Flavors, err = c.getFlavors(); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func (c *client) getRouters() ([]*models.Router, error) {
	result := []*models.Router{}

	err := routers.List(c.NetworkClient, routers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		if routerList, err := routers.ExtractRouters(page); err != nil {
			return false, err
		} else {
			for _, router := range routerList {
				result = append(result, &models.Router{ID: router.ID, Name: router.Name})
			}
		}
		return true, nil
	})

	if err != nil {
		return result, err
	}

	for _, router := range result {
		if router.Networks, err = c.getNetworks(router); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (c *client) getNetworks(router *models.Router) ([]*models.Network, error) {
	result := []*models.Network{}

	networkIDs, err := c.getRouterNetworkIDs(router)
	if err != nil {
		return result, err
	}

	for _, networkID := range networkIDs {
		network, err := networks.Get(c.NetworkClient, networkID).Extract()
		if err != nil {
			return result, err
		}
		result = append(result, &models.Network{ID: network.ID, Name: network.Name})
	}

	for _, network := range result {
		if network.Subnets, err = c.getSubnets(network); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *client) getRouterNetworkIDs(router *models.Router) ([]string, error) {
	result := []string{}

	err := ports.List(c.NetworkClient, ports.ListOpts{DeviceID: router.ID, DeviceOwner: "network:router_interface"}).EachPage(func(page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, port := range portList {
			result = append(result, port.NetworkID)
		}
		return true, nil
	})

	return result, err
}

func (c *client) getSubnetIDs(network *models.Network) ([]string, error) {
	result, err := networks.Get(c.NetworkClient, network.ID).Extract()
	if err != nil {
		return []string{}, err
	}

	return result.Subnets, nil
}

func (c *client) getSubnets(network *models.Network) ([]*models.Subnet, error) {
	result := []*models.Subnet{}

	subnetIDs, err := c.getSubnetIDs(network)
	if err != nil {
		return result, err
	}

	for _, subnetID := range subnetIDs {
		subnet, err := subnets.Get(c.NetworkClient, subnetID).Extract()
		if err != nil {
			return result, err
		}
		result = append(result, &models.Subnet{ID: subnet.ID, Name: subnet.Name, CIDR: subnet.CIDR})
	}

	return result, nil
}

func (c *client) getKeyPairs() ([]*models.KeyPair, error) {
	result := []*models.KeyPair{}

	pager, err := keypairs.List(c.ComputeClient).AllPages()
	if err != nil {
		return result, err
	}

	keyList, err := keypairs.ExtractKeyPairs(pager)
	if err != nil {
		return result, err
	}

	for _, key := range keyList {
		result = append(result, &models.KeyPair{Name: key.Name})
	}

	return result, nil
}

func (c *client) getSecurityGroups() ([]*models.SecurityGroup, error) {
	result := []*models.SecurityGroup{}

	err := secgroups.List(c.ComputeClient).EachPage(func(page pagination.Page) (bool, error) {
		secGroupList, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			return false, err
		}
		for _, secGroup := range secGroupList {
			result = append(result, &models.SecurityGroup{ID: secGroup.ID, Name: secGroup.Name})
		}
		return true, nil
	})

	return result, err
}

func (c *client) getFlavors() ([]*models.Flavor, error) {
	result := []*models.Flavor{}

	err := flavors.ListDetail(c.ComputeClient, &flavors.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		list, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		for _, entry := range list {
			result = append(result, &models.Flavor{ID: entry.ID, Name: entry.Name})
		}
		return true, nil
	})

	return result, err
}
