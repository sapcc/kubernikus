package admin

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
)

type LoggingClient struct {
	Client AdminClient
	Logger log.Logger
}

func (c LoggingClient) CreateKlusterServiceUser(username, password, domainName, projectID string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "created kluster service user",
			"username", username,
			"domain_name", domainName,
			"project_id", projectID,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.CreateKlusterServiceUser(username, password, domainName, projectID)
}

func (c LoggingClient) DeleteUser(username, domainName string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "deleted kluster service user",
			"username", username,
			"domain_name", domainName,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.DeleteUser(username, domainName)
}

func (c LoggingClient) GetKubernikusCatalogEntry() (url string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "retrieved kubernikus url from catalog",
			"url", url,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetKubernikusCatalogEntry()
}

func (c LoggingClient) GetRegion() (region string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "retrieved region",
			"region", region,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetRegion()
}

func (c LoggingClient) CreateStorageContainer(projectID, containerName, serviceUserName, serviceUserDomainName string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "create storage container",
			"project_id", projectID,
			"container_name", containerName,
			"service_user_name", serviceUserName,
			"service_user_domain", serviceUserDomainName,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.CreateStorageContainer(projectID, containerName, serviceUserName, serviceUserDomainName)
}

func (c LoggingClient) GetStorageContainerMeta(projectID, containerName string) (result *ContainerMeta, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "checking if storage container exists",
			"project_id", projectID,
			"container_name", containerName,
			"took", time.Since(begin),
			"meta", result,
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetStorageContainerMeta(projectID, containerName)
}

func (c LoggingClient) GetContainerACLEntry(projectID, serviceUserName, serviceUserDomainName string) (result string, err error) {
	return c.Client.GetContainerACLEntry(projectID, serviceUserName, serviceUserDomainName)
}

func (c LoggingClient) UpdateStorageContainerMeta(projectID, container string, meta ContainerMeta) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "updating storage container",
			"project_id", projectID,
			"container_name", container,
			"took", time.Since(begin),
			"meta", meta,
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.UpdateStorageContainerMeta(projectID, container, meta)
}

func (c LoggingClient) AssignUserRoles(projectID, userName, domainName string, userRoles []string) (err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "assign user roles",
			"project_id", projectID,
			"user_id", userName,
			"domain_name", domainName,
			"user_roles", fmt.Sprintf("%v", userRoles),
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.AssignUserRoles(projectID, userName, domainName, userRoles)
}

func (c LoggingClient) GetUserRoles(projectID, userName, domainName string) (userRoles []string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "get user roles",
			"project_id", projectID,
			"user_id", userName,
			"domain_name", domainName,
			"roles", fmt.Sprintf("%v", userRoles),
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetUserRoles(projectID, userName, domainName)
}

func (c LoggingClient) GetDomainNameByProject(projectID string) (domainName string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "get domain name by project",
			"project_id", projectID,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetDomainNameByProject(projectID)
}

func (c LoggingClient) GetDefaultServiceUserRoles() (roles []string) {
	return c.Client.GetDefaultServiceUserRoles()
}

func (c LoggingClient) GetDomainID(domainName string) (domainId string, err error) {
	defer func(begin time.Time) {
		c.Logger.Log(
			"msg", "get domain id by domain name",
			"domain_name", domainName,
			"took", time.Since(begin),
			"v", 2,
			"err", err,
		)
	}(time.Now())
	return c.Client.GetDomainID(domainName)
}
