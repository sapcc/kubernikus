package util

import (
	"github.com/gophercloud/gophercloud"
)

type AuthenticatedClient struct {
	providerClient *gophercloud.ProviderClient
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
}
