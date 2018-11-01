package admin

import (
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
