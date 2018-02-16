package main

import (
	"fmt"
	"os"

	"strings"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

type OpenStackCredentials struct {
	ProjectName       string `yaml:"project_name"`
	ProjectDomainName string `yaml:"project_domain_name"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	UserDomainName    string `yaml:"user_domain_name"`
	AuthURL           string `yaml:"auth_url"`
	RegionName        string `yaml:"region_name"`
	Token             string `yaml:"token"`
}

func newOpenStackServiceClient(authURL, region string) (*gophercloud.ServiceClient, error) {
	provider, err := openstack.NewClient(authURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize openstack client: %v", err)
	}
	return &gophercloud.ServiceClient{
		ProviderClient: provider,
		Endpoint:       authURL,
	}, nil
}

func getOpenStackCredentialsFromENV() OpenStackCredentials {
	osToken := os.Getenv("OS_TOKEN")
	if osToken != "" {
		return OpenStackCredentials{
			AuthURL:    os.Getenv("OS_AUTH_URL"),
			RegionName: os.Getenv("OS_REGION_NAME"),
			Token:      os.Getenv(osToken),
		}
	}
	return OpenStackCredentials{
		AuthURL:           os.Getenv("OS_AUTH_URL"),
		RegionName:        os.Getenv("OS_REGION_NAME"),
		Username:          os.Getenv("OS_USERNAME"),
		Password:          os.Getenv("OS_PASSWORD"),
		UserDomainName:    os.Getenv("OS_USER_DOMAIN_NAME"),
		ProjectName:       os.Getenv("OS_PROJECT_NAME"),
		ProjectDomainName: os.Getenv("OS_PROJECT_DOMAIN_NAME"),
	}
}

func getToken(c OpenStackCredentials) (string, error) {
	opts := &tokens.AuthOptions{
		IdentityEndpoint: c.AuthURL,
		Username:         c.Username,
		Password:         c.Password,
		DomainName:       c.UserDomainName,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: c.ProjectName,
			DomainName:  c.ProjectDomainName,
		},
	}

	client, err := newOpenStackServiceClient(
		c.AuthURL,
		c.RegionName,
	)
	if err != nil {
		return "", err
	}

	token, err := tokens.Create(client, opts).ExtractToken()
	if err != nil {
		return "", err
	}
	glog.Infof("Obtained Keystone token")

	return token.ID, nil
}

// Verify parameters used for OpenStack authentication
func (c *OpenStackCredentials) Verify() error {
	errorString := ""
	if c.ProjectName == "" {
		errorString += "missing OS_PROJECT_NAME\n"
	}
	if c.ProjectDomainName == "" {
		errorString += "missing OS_PROJECT_DOMAIN_NAME\n"
	}
	if c.Username == "" {
		errorString += "missing OS_USERNAME\n"
	}
	if c.UserDomainName == "" {
		errorString += "missing OS_USER_DOMAIN_NAME\n"
	}
	if c.Password == "" {
		errorString += "missing OS_PASSWORD\n"
	}
	if c.AuthURL == "" {
		errorString += "missing OS_AUTH_URL\n"
	} else {
		if !strings.HasSuffix(c.AuthURL, "/") {
			c.AuthURL += "/"
		}
	}
	if errorString != "" {
		return fmt.Errorf(errorString)
	}
	return nil
}

func (s *E2ETestSuite) authFunc() runtime.ClientAuthInfoWriterFunc {
	return runtime.ClientAuthInfoWriterFunc(
		func(req runtime.ClientRequest, reg strfmt.Registry) error {
			req.SetHeaderParam("X-AUTH-TOKEN", s.Token)
			return nil
		})
}
