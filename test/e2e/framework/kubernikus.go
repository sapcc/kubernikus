package framework

import (
	"fmt"
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	kubernikus "github.com/sapcc/kubernikus/pkg/api/client"
)

type Kubernikus struct {
	Client   *kubernikus.Kubernikus
	AuthInfo runtime.ClientAuthInfoWriterFunc
}

func NewKubernikusFramework(host string) (*Kubernikus, error) {
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

	if err := openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{}); err != nil {
		return nil, fmt.Errorf("could not authenticate with openstack: %v", err)
	}

	authInfo := runtime.ClientAuthInfoWriterFunc(
		func(req runtime.ClientRequest, reg strfmt.Registry) error {
			req.SetHeaderParam("X-AUTH-TOKEN", provider.Token())
			return nil
		})

	kubernikusClient := kubernikus.NewHTTPClientWithConfig(
		nil,
		&kubernikus.TransportConfig{
			Host:    host,
			Schemes: []string{"https"},
		},
	)

	return &Kubernikus{
		Client:   kubernikusClient,
		AuthInfo: authInfo,
	}, nil
}
