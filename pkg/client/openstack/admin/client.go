package admin

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	gc_roles "github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
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
	CreateStorageContainer(projectID, containerName, serviceUserName, serviceUserDomainName string) error
	AssignUserRoles(string, string, string, []string) error
	GetUserRoles(string, string, string) ([]string, error)
	GetDefaultServiceUserRoles() []string
}

type adminClient struct {
	ProviderClient *gophercloud.ProviderClient
	IdentityClient *gophercloud.ServiceClient

	domainNameToID sync.Map
	roleNameToID   sync.Map
}

func NewAdminClient(providerClient *gophercloud.ProviderClient) (AdminClient, error) {
	var client AdminClient

	identity, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	client = &adminClient{
		ProviderClient: providerClient,
		IdentityClient: identity,
	}

	return client, nil
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
	description := "Kubernikus kluster service user"
	if user != nil {
		user, err = users.Update(c.IdentityClient, user.ID, users.UpdateOpts{
			Password:         password,
			DefaultProjectID: projectID,
			Description:      &description,
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

	err = c.AssignUserRoles(projectID, username, domainName, c.GetDefaultServiceUserRoles())
	if err != nil {
		return fmt.Errorf("Failed to assign roles to service user: %s", err)
	}

	return nil
}

func (c *adminClient) AssignUserRoles(projectID, userName, domainName string, userRoles []string) error {
	domainID, err := c.getDomainID(domainName)
	if err != nil {
		return err
	}

	user, err := c.getUserByName(userName, domainID)
	if err != nil {
		return err
	}

	for _, roleName := range userRoles {
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

func (c *adminClient) GetUserRoles(projectID, userName, domainName string) ([]string, error) {
	domainID, err := c.getDomainID(domainName)
	if err != nil {
		return nil, err
	}

	user, err := c.getUserByName(userName, domainID)
	if err != nil {
		return nil, err
	}

	listOpts := gc_roles.ListAssignmentsOnResourceOpts{
		UserID:    user.ID,
		ProjectID: projectID,
	}

	pages, err := gc_roles.ListAssignmentsOnResource(c.IdentityClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	userRoles, err := gc_roles.ExtractRoles(pages)
	if err != nil {
		return nil, err
	}

	var retRoles []string
	for _, role := range userRoles {
		retRoles = append(retRoles, role.Name)
	}

	return retRoles, nil
}

func (c *adminClient) GetDefaultServiceUserRoles() []string {
	return serviceUserRoles
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

func (c *adminClient) CreateStorageContainer(projectID, containerName, serviceUserName, serviceUserDomainName string) error {
	endpointURL, err := c.getPublicObjectStoreEndpointURL(projectID)
	if err != nil {
		return err
	}

	storageClient, err := openstack.NewObjectStorageV1(c.ProviderClient, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}
	storageClient.Endpoint = endpointURL

	domainID, err := c.getDomainID(serviceUserDomainName)
	if err != nil {
		return err
	}

	serviceUser, err := c.getUserByName(serviceUserName, domainID)
	if err != nil {
		return err
	}

	acl := fmt.Sprintf("%s:%s", projectID, serviceUser.ID)
	createOpts := containers.CreateOpts{
		ContainerRead:  acl,
		ContainerWrite: acl,
	}

	_, err = containers.Create(storageClient, containerName, createOpts).Extract()

	return err
}

func (c *adminClient) getPublicObjectStoreEndpointURL(projectID string) (string, error) {
	serviceListOpts := services.ListOpts{
		ServiceType: "object-store",
	}

	allServicesPages, err := services.List(c.IdentityClient, serviceListOpts).AllPages()
	if err != nil {
		return "", err
	}

	allServices, err := services.ExtractServices(allServicesPages)
	if err != nil {
		return "", err
	}
	if len(allServices) != 1 {
		return "", errors.New("only one service expected")
	}

	endpointListOpts := endpoints.ListOpts{
		ServiceID:    allServices[0].ID,
		Availability: gophercloud.AvailabilityPublic,
	}

	allEndpointPages, err := endpoints.List(c.IdentityClient, endpointListOpts).AllPages()
	if err != nil {
		return "", err
	}

	allEndpoints, err := endpoints.ExtractEndpoints(allEndpointPages)
	if err != nil {
		return "", err
	}
	if len(allEndpoints) != 1 {
		return "", errors.New("only one endpoint expected")
	}

	endpointURL := strings.Replace(allEndpoints[0].URL, "%(tenant_id)s", projectID, 1)
	return gophercloud.NormalizeURL(endpointURL), nil
}
