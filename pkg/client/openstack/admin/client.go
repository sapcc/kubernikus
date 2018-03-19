package admin

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/sapcc/kubernikus/pkg/client/openstack/domains"
	"github.com/sapcc/kubernikus/pkg/client/openstack/roles"
)

var serviceUserRoles = []string{"network_admin", "member"}

type AdminClient interface {
	CreateKlusterServiceUser(username, password, domainName, projectID string) error
	DeleteUser(username, domainName string) error
	GetKubernikusCatalogEntry() (string, error)
	GetRegion() (string, error)
}

type adminClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient

	domainNameToID sync.Map
	roleNameToID   sync.Map
}

func NewAdminClient(network, compute, identity *gophercloud.ServiceClient) AdminClient {
	var client AdminClient
	client = &adminClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
	}
	return client
}

func (c *adminClient) CreateKlusterServiceUser(username, password, domainName, projectID string) error {
	domainID, err := c.getDomainID(domainName)
	if err != nil {
		return err
	}

	user, err := c.getUserByName(username, domainID)
	if err != nil {
		return err
	}

	//Do we need to update or create?
	if user != nil {
		user, err = users.Update(c.IdentityClient, user.ID, users.UpdateOpts{
			Password:         password,
			DefaultProjectID: projectID,
			Description:      "Kubernikus kluster service user",
		}).Extract()
	} else {
		user, err = users.Create(c.IdentityClient, users.CreateOpts{
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
		roleID, err := c.getRoleID(roleName)
		if err != nil {
			return err
		}
		err = roles.AssignToUserInProject(c.IdentityClient, projectID, user.ID, roleID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *adminClient) DeleteUser(username, domainName string) error {
	domainID, err := c.getDomainID(domainName)
	if err != nil {
		return err
	}
	return users.List(c.IdentityClient, users.ListOpts{DomainID: domainID, Name: username}).EachPage(func(page pagination.Page) (bool, error) {
		userList, err := users.ExtractUsers(page)
		if err != nil {
			return false, err
		}
		switch len(userList) {
		case 0:
			return false, nil
		case 1:
			return false, users.Delete(c.IdentityClient, userList[0].ID).ExtractErr()
		default:
			return false, errors.New("Multiple users found")
		}
	})
}

func (c *adminClient) GetRegion() (string, error) {
	opts := services.ListOpts{ServiceType: "compute"}
	computeServiceID := ""
	err := services.List(c.IdentityClient, opts).EachPage(func(page pagination.Page) (bool, error) {
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
	err = endpoints.List(c.IdentityClient, endpointOpts).EachPage(func(page pagination.Page) (bool, error) {
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

func (c *adminClient) GetKubernikusCatalogEntry() (string, error) {
	catalog, err := tokens.Get(c.IdentityClient, c.IdentityClient.ProviderClient.TokenID).ExtractServiceCatalog()
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

func (c *adminClient) getDomainID(domainName string) (string, error) {
	if id, ok := c.domainNameToID.Load(domainName); ok {
		return id.(string), nil
	}
	err := domains.List(c.IdentityClient, &domains.ListOpts{Name: domainName}).EachPage(func(page pagination.Page) (bool, error) {
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
			return false, errors.New("More than one domain found")
		}
	})
	if err != nil {
		return "", err
	}

	id, _ := c.domainNameToID.Load(domainName)
	return id.(string), nil
}

func (c *adminClient) getUserByName(username, domainID string) (*users.User, error) {
	var user *users.User
	err := users.List(c.IdentityClient, users.ListOpts{DomainID: domainID, Name: username}).EachPage(func(page pagination.Page) (bool, error) {
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

func (c *adminClient) getRoleID(roleName string) (string, error) {
	if id, ok := c.roleNameToID.Load(roleName); ok {
		return id.(string), nil
	}
	err := roles.List(c.IdentityClient, nil).EachPage(func(page pagination.Page) (bool, error) {
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
