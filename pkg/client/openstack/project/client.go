package project

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	securitygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

type ProjectClient interface {
	GetMetadata() (*models.OpenstackMetadata, error)
}

type projectClient struct {
	projectID string

	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
	StorageClient  *gophercloud.ServiceClient
}

func NewProjectClient(projectID string, network, compute, identity *gophercloud.ServiceClient) ProjectClient {
	var client ProjectClient
	client = &projectClient{
		projectID:      projectID,
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
	}
	return client
}

func (c *projectClient) GetMetadata() (metadata *models.OpenstackMetadata, err error) {
	metadata = &models.OpenstackMetadata{
		Flavors:        make([]models.Flavor, 0),
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

func (c *projectClient) getRouters() ([]*models.Router, error) {
	result := []*models.Router{}

	err := routers.List(c.NetworkClient, routers.ListOpts{TenantID: c.projectID}).EachPage(func(page pagination.Page) (bool, error) {
		if routerList, err := routers.ExtractRouters(page); err != nil {
			return false, err
		} else {
			for _, router := range routerList {
				result = append(result, &models.Router{ID: router.ID, Name: router.Name, ExternalNetworkID: router.GatewayInfo.NetworkID})
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

func (c *projectClient) getNetworks(router *models.Router) ([]*models.Network, error) {
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

func (c *projectClient) getRouterNetworkIDs(router *models.Router) ([]string, error) {
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

func (c *projectClient) getSubnetIDs(network *models.Network) ([]string, error) {
	result, err := networks.Get(c.NetworkClient, network.ID).Extract()
	if err != nil {
		return []string{}, err
	}

	return result.Subnets, nil
}

func (c *projectClient) getSubnets(network *models.Network) ([]*models.Subnet, error) {
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

func (c *projectClient) getKeyPairs() ([]*models.KeyPair, error) {
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
		result = append(result, &models.KeyPair{Name: key.Name, PublicKey: key.PublicKey})
	}

	return result, nil
}

func (c *projectClient) getSecurityGroups() ([]*models.SecurityGroup, error) {
	result := []*models.SecurityGroup{}

	err := securitygroups.List(c.NetworkClient, securitygroups.ListOpts{TenantID: c.projectID}).EachPage(func(page pagination.Page) (bool, error) {
		secGroupList, err := securitygroups.ExtractGroups(page)
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

func (c *projectClient) getFlavors() ([]models.Flavor, error) {
	result := []models.Flavor{}

	err := flavors.ListDetail(c.ComputeClient, &flavors.ListOpts{MinRAM: 2000}).EachPage(func(page pagination.Page) (bool, error) {
		list, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		for _, entry := range list {
			result = append(result, models.Flavor{ID: entry.ID, Name: entry.Name, RAM: int64(entry.RAM), Vcpus: int64(entry.VCPUs)})
		}
		return true, nil
	})
	models.SortFlavors(result)

	return result, err
}
