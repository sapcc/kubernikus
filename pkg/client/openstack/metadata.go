package openstack

import (
	"net/http"

	"github.com/golang/glog"
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
)

type scopedClient struct {
	providerClient *gophercloud.ProviderClient
	networkClient  *gophercloud.ServiceClient
	computeClient  *gophercloud.ServiceClient
	identityClient *gophercloud.ServiceClient
}

type ScopedClient interface {
	GetMetadata() (*models.OpenstackMetadata, error)
}

type CustomRoundTripper struct {
	rt http.RoundTripper
}

func (crt *CustomRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := crt.rt.RoundTrip(request)
	if response == nil {
		return nil, err
	}

	for name, header := range response.Header {
		for _, v := range header {
			glog.Infof("%v: %v", name, v)
		}
	}

	return response, err
}

func NewScopedClient(authOptions *tokens.AuthOptions) (ScopedClient, error) {
	var err error
	client := &scopedClient{}

	if client.providerClient, err = openstack.NewClient(authOptions.IdentityEndpoint); err != nil {
		return nil, err
	}

	transport := client.providerClient.HTTPClient.Transport
	client.providerClient.HTTPClient.Transport = &CustomRoundTripper{
		rt: transport,
	}

	if err := openstack.AuthenticateV3(client.providerClient, authOptions, gophercloud.EndpointOpts{}); err != nil {
		return nil, err
	}

	if client.identityClient, err = openstack.NewIdentityV3(client.providerClient, gophercloud.EndpointOpts{}); err != nil {
		return nil, err
	}

	if client.computeClient, err = openstack.NewComputeV2(client.providerClient, gophercloud.EndpointOpts{}); err != nil {
		return nil, err
	}

	if client.networkClient, err = openstack.NewNetworkV2(client.providerClient, gophercloud.EndpointOpts{}); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *scopedClient) GetMetadata() (*models.OpenstackMetadata, error) {
	var err error
	metadata := &models.OpenstackMetadata{}

	if metadata.Routers, err = c.getRouters(); err != nil {
		return nil, err
	}

	if metadata.KeyPairs, err = c.getKeyPairs(); err != nil {
		return nil, err
	}

	if metadata.SecurityGroups, err = c.getSecurityGroups(); err != nil {
		return nil, err
	}

	if metadata.Flavors, err = c.getFlavors(); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (c *scopedClient) getRouters() ([]*models.Router, error) {
	result := []*models.Router{}

	pager := routers.List(c.networkClient, routers.ListOpts{})

	for _, k := range pager.Headers {
		glog.Infof("%s: %s ", k, pager.Headers[k])
	}

	pager = pager.AllPages()
		for _, k := range pager.Headers {
			glog.Infof("%s: %s ", k, pager.Headers[k])
		}
		return SinglePageBase: pagination.SinglePageBase(r)
	})

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		glog.Infof("%s", page.GetBody())

		if routerList, err := routers.ExtractRouters(page); err != nil {
			return false, err
		} else {
			for _, router := range routerList {
				result = append(result, &models.Router{ID: router.ID, Name: router.Name})
			}
		}
		return true, nil
	})

	for _, k := range pager.Headers {
		glog.Infof("%s: %s ", k, pager.Headers[k])
	}

	if err != nil {
		return nil, err
	}

	for _, router := range result {
		if router.Networks, err = c.getNetworks(router); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *scopedClient) getNetworks(router *models.Router) ([]*models.Network, error) {
	result := []*models.Network{}

	networkIDs, err := c.getRouterNetworkIDs(router)
	if err != nil {
		return nil, err
	}

	for _, networkID := range networkIDs {
		network, err := networks.Get(c.networkClient, networkID).Extract()
		if err != nil {
			return nil, err
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

func (c *scopedClient) getRouterNetworkIDs(router *models.Router) ([]string, error) {
	result := []string{}

	err := ports.List(c.networkClient, ports.ListOpts{DeviceID: router.ID, DeviceOwner: "network:router_interface"}).EachPage(func(page pagination.Page) (bool, error) {
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

func (c *scopedClient) getSubnetIDs(network *models.Network) ([]string, error) {
	result, err := networks.Get(c.networkClient, network.ID).Extract()
	if err != nil {
		return nil, err
	}

	return result.Subnets, nil
}

func (c *scopedClient) getSubnets(network *models.Network) ([]*models.Subnet, error) {
	result := []*models.Subnet{}

	subnetIDs, err := c.getSubnetIDs(network)
	if err != nil {
		return nil, err
	}

	for _, subnetID := range subnetIDs {
		subnet, err := subnets.Get(c.networkClient, subnetID).Extract()
		if err != nil {
			return nil, err
		}
		result = append(result, &models.Subnet{ID: subnet.ID, Name: subnet.Name, CIDR: subnet.CIDR})
	}

	return result, nil
}

func (c *scopedClient) getKeyPairs() ([]*models.KeyPair, error) {
	result := []*models.KeyPair{}

	pager, err := keypairs.List(c.computeClient).AllPages()
	if err != nil {
		return nil, err
	}

	keyList, err := keypairs.ExtractKeyPairs(pager)
	if err != nil {
		return nil, err
	}

	for _, key := range keyList {
		result = append(result, &models.KeyPair{Name: key.Name})
	}

	return result, nil
}

func (c *scopedClient) getSecurityGroups() ([]*models.SecurityGroup, error) {
	result := []*models.SecurityGroup{}

	err := secgroups.List(c.computeClient).EachPage(func(page pagination.Page) (bool, error) {
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

func (c *scopedClient) getFlavors() ([]*models.Flavor, error) {
	result := []*models.Flavor{}

	err := flavors.ListDetail(c.computeClient, &flavors.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
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
