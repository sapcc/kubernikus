package common

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type OpenstackClient struct {
	*tokens.AuthOptions
	Provider *gophercloud.ProviderClient
	Identity *gophercloud.ServiceClient
}

func NewOpenstackClient() *OpenstackClient {
	return &OpenstackClient{
		&tokens.AuthOptions{
			IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
			Username:         os.Getenv("OS_USERNAME"),
			UserID:           os.Getenv("OS_USER_ID"),
			Password:         os.Getenv("OS_PASSWORD"),
			DomainID:         os.Getenv("OS_USER_DOMAIN_ID"),
			DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
			AllowReauth:      true,
			Scope: tokens.Scope{
				ProjectID:   os.Getenv("OS_PROJECT_ID"),
				ProjectName: os.Getenv("OS_PROJECT_NAME"),
				DomainID:    os.Getenv("OS_PROJECT_DOMAIN_ID"),
				DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
			},
		}, nil, nil,
	}
}

func (o *OpenstackClient) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.IdentityEndpoint, "auth-url", o.IdentityEndpoint, "Openstack Keystone Endpoint URL [OS_AUTH_URL]")
	flags.StringVar(&o.UserID, "user-id", o.UserID, "User ID [OS_USER_ID]")
	flags.StringVar(&o.Username, "username", o.Username, "User name. Also requires --user-domain-name/--user-domain-id [OS_USERNAME]")
	flags.StringVar(&o.Password, "password", "", "User password [OS_PASSWORD]")
	flags.StringVar(&o.DomainID, "user-domain-id", o.DomainID, "User domain [OS_USER_DOMAIN_ID]")
	flags.StringVar(&o.DomainName, "user-domain-name", o.DomainName, "User domain [OS_USER_DOMAIN_NAME]")
	flags.StringVar(&o.Scope.ProjectID, "project-id", o.Scope.ProjectID, "Scope to this project [OS_PROJECT_ID]")
	flags.StringVar(&o.Scope.ProjectName, "project-name", o.Scope.ProjectName, "Scope to this project. Also requires --project-domain-name/--project-domain-id [OS_PROJECT_NAME]")
	flags.StringVar(&o.Scope.DomainID, "project-domain-id", o.Scope.DomainID, "Domain of the project [OS_PROJECT_DOMAIN_ID]")
	flags.StringVar(&o.Scope.DomainName, "project-domain-name", o.Scope.DomainName, "Domain of the project [OS_PROJECT_DOMAIN_NAME]")
}

func (o *OpenstackClient) Validate(c *cobra.Command, args []string) error {
	if o.IdentityEndpoint == "" {
		return errors.Errorf("You need to provide --auth-url or OS_AUTH_URL")
	} else {
		if _, err := url.Parse(o.IdentityEndpoint); err != nil {
			return errors.Errorf("The URL for the Kubernikus API is not parsable")
		}
	}

	if o.Username == "" {
		if o.UserID == "" {
			return errors.Errorf("You need to provide --username/--user-id or OS_USERNAME/OS_USER_ID")
		}
	} else {
		if o.DomainName == "" && o.DomainID == "" {
			return errors.Errorf("You need to provide --user-domain-name/--user-domain-id or OS_USER_DOMAIN_NAME/OS_USER_DOMAIN_ID")
		}
	}

	if o.Scope.ProjectName == "" {
		if o.Scope.ProjectID == "" {
			return errors.Errorf("You need to provide --project-name/--project-id or OS_PROJECT_NAME/OS_PROJECT_ID")
		}
	} else {
		if o.Scope.DomainName == "" && o.DomainID == "" {
			return errors.Errorf("You need to provide --project-domain-name/--project-domain-id or OS_PROJECT_DOMAIN_NAME/OS_PROJECT_DOMAIN_ID")
		}
	}

	return nil
}

func (o *OpenstackClient) Complete(args []string) error {
	if err := o.Setup(); err != nil {
		return err
	}

	return nil
}

func (o *OpenstackClient) Setup() error {
	var err error

	if o.Password == "" {
		o.Password = os.Getenv("OS_PASSWORD")
	}

	if o.Provider, err = openstack.NewClient(o.IdentityEndpoint); err != nil {
		return errors.Wrap(err, "Creating Gophercloud ProviderClient failed")
	}

	if o.Identity, err = openstack.NewIdentityV3(o.Provider, gophercloud.EndpointOpts{}); err != nil {
		return errors.Wrap(err, "Creating Identity ServiceClient failed")
	}

	return nil
}

func (o *OpenstackClient) PrintAuthInfo() string {
	var user, scope string

	if o.UserID != "" {
		user = o.UserID
	} else {
		if o.DomainID != "" {
			user = fmt.Sprintf("%v/%v", o.DomainID, o.Username)
		} else {
			user = fmt.Sprintf("%v/%v", o.DomainName, o.Username)
		}
	}

	if o.Scope.ProjectID != "" {
		scope = o.Scope.ProjectID
	} else {
		if o.Scope.DomainID != "" {
			scope = fmt.Sprintf("%v/%v", o.Scope.DomainID, o.Scope.ProjectName)
		} else {
			scope = fmt.Sprintf("%v/%v", o.Scope.DomainName, o.Scope.ProjectName)
		}
	}

	return fmt.Sprintf("Authenticating %v at %v", user, scope)
}

func (o *OpenstackClient) PrintDebugAuthInfo() string {
	funcMap := template.FuncMap{
		"mask": func(input string) string {
			return strings.Repeat("*", len(input))
		},
	}

	tmpl := `Using AuthInfo:
    IdentityEndpoint: {{ .IdentityEndpoint }}
    Username:         {{ .Username }}
    UserID:           {{ .UserID }}
    Password:         {{ mask .Password }}
    DomainID:         {{ .DomainID }}
    DomainName:       {{ .DomainName }}
    Scope:
      ProjectID:      {{ .Scope.ProjectID }}
      ProjectName:    {{ .Scope.ProjectName }}
      DomainID:       {{ .Scope.DomainID }}
      DomainName:     {{ .Scope.DomainName }}`

	t := template.Must(template.New("t").Funcs(funcMap).Parse(tmpl))
	var output bytes.Buffer
	if err := t.Execute(&output, o); err != nil {
		return err.Error()
	}

	return output.String()

}

func (o *OpenstackClient) Authenticate() error {
	return openstack.AuthenticateV3(o.Provider, o, gophercloud.EndpointOpts{})
}

func (o *OpenstackClient) DefaultKubernikusURL() (*url.URL, error) {
	catalog, err := tokens.Create(o.Identity, o).ExtractServiceCatalog()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't fetch service catalog")
	}

	result := ""
	for _, service := range catalog.Entries {
		if service.Type == "kubernikus" {
			for _, endpoint := range service.Endpoints {
				if endpoint.Interface == "public" {
					result = endpoint.URL
				}
			}
		}
	}

	if result == "" {
		return nil, errors.Errorf("No public Kubernikus service found in the service catalog")
	}

	url, err := url.Parse(result)
	if err != nil {
		return nil, errors.Wrapf(err, "The URL for the Kubernikus API is not parsable")
	}

	return url, nil
}
