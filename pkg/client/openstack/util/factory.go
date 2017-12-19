package util

import (
	"github.com/gophercloud/gophercloud"
)

type AuthenticatedClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
}
