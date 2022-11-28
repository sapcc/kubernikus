package framework

import (
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

type OpenStack struct {
	Provider     *gophercloud.ProviderClient
	Compute      *gophercloud.ServiceClient
	Identity     *gophercloud.ServiceClient
	BlockStorage *gophercloud.ServiceClient
	Network      *gophercloud.ServiceClient
}

func NewOpenStackFramework() (*OpenStack, error) {
	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: os.Getenv("OS_PROJECT_NAME"),
			DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		},
	}

	provider, err := openstack.NewClient(os.Getenv("OS_AUTH_URL"))
	if err != nil {
		return nil, fmt.Errorf("could not initialize openstack client: %v", err)
	}

	provider.UseTokenLock()

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not authenticat provider client: %v", err)
	}

	identity, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not initialize identity client: %v", err)
	}

	compute, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	compute.Microversion = "2.52"
	if err != nil {
		return nil, fmt.Errorf("could not initialize compute client: %v", err)
	}

	blockStorage, err := openstack.NewBlockStorageV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not initialize blockStorage client: %v", err)
	}

	network, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not initialize network client: %v", err)
	}

	return &OpenStack{
		Provider:     provider,
		Compute:      compute,
		Identity:     identity,
		BlockStorage: blockStorage,
		Network:      network,
	}, nil
}
