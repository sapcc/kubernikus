package kubernikusctl

import (
	"fmt"
	"net/url"
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	kubernikus "github.com/sapcc/kubernikus/pkg/client/kubernikus_generated"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus_generated/operations"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func NewCredentialsCommand() *cobra.Command {
	o := NewCredentialsOptions()

	c := &cobra.Command{
		Use:   "credentials [name]",
		Short: "Fetches Kubernikus credentials via API",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type CredentialsOptions struct {
	url  string
	name string

	kubernikus *kubernikus.Kubernikus
	auth       *tokens.AuthOptions
	provider   *gophercloud.ProviderClient
	identity   *gophercloud.ServiceClient
	kubeconfig *clientcmdapi.Config
}

func NewCredentialsOptions() *CredentialsOptions {
	username := os.Getenv("OS_USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}

	o := &CredentialsOptions{
		name: os.Getenv("KUBERNIKUS_NAME"),
		url:  os.Getenv("KUBERNIKUS_URL"),
	}
	o.auth = &tokens.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         username,
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
	}

	return o
}

func (o *CredentialsOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.auth.IdentityEndpoint, "auth-url", o.auth.IdentityEndpoint, "Openstack keystone url [OS_AUTH_URL]")
	flags.StringVar(&o.auth.UserID, "user-id", o.auth.UserID, "User ID [OS_USER_ID]")
	flags.StringVar(&o.auth.Username, "username", o.auth.Username, "User name. Also requires --user-domain-name/--user-domain-id [OS_USERNAME]")
	flags.StringVar(&o.auth.Password, "password", o.auth.Password, "User password [OS_PASSWORD]")
	flags.StringVar(&o.auth.DomainID, "user-domain-id", o.auth.DomainID, "User domain [OS_USER_DOMAIN_ID]")
	flags.StringVar(&o.auth.DomainName, "user-domain-name", o.auth.DomainName, "User domain [OS_USER_DOMAIN_NAME]")
	flags.StringVar(&o.auth.Scope.ProjectID, "project-id", o.auth.Scope.ProjectID, "Scope to this project [OS_PROJECT_ID]")
	flags.StringVar(&o.auth.Scope.ProjectName, "project-name", o.auth.Scope.ProjectName, "Scope to this project. Also requires --project-domain-name/--project-domain-id [OS_PROJECT_NAME]")
	flags.StringVar(&o.auth.Scope.DomainID, "project-domain-id", o.auth.Scope.DomainID, "Domain of the project [OS_PROJECT_DOMAIN_ID]")
	flags.StringVar(&o.auth.Scope.DomainName, "project-domain-name", o.auth.Scope.DomainName, "Domain of the project [OS_PROJECT_DOMAIN_NAME]")
	flags.StringVar(&o.url, "url", o.url, "URL for Kubernikus API")
	flags.StringVar(&o.name, "name", o.name, "Cluster Name")
}

func (o *CredentialsOptions) Validate(c *cobra.Command, args []string) error {
	if o.auth.IdentityEndpoint == "" {
		return errors.Errorf("You need to provide --auth-url or OS_AUTH_URL")
	}

	if o.auth.Username == "" && o.auth.UserID  == "" {
		return errors.Errorf("You need to provide --username/--user-id or OS_USERNAME/OS_USER_ID")
	}

	if o.auth.Password == "" {
		return errors.Errorf("You need to provide --password or OS_PASSWORD")
	}

	if o.auth.Username != "" && o.auth.UserID == "" && o.auth.DomainName == "" && o.auth.DomainID == "" {
		return errors.Errorf("You need to provide --user-domain-name/--user-domain-id or OS_USER_DOMAIN_NAME/OS_USER_DOMAIN_ID")
	}

	if o.auth.Scope.ProjectName == "" && o.auth.Scope.ProjectID == "" {
		return errors.Errorf("You need to provide --project-name/--project-id or OS_PROJECT_NAME/OS_PROJECT_ID")
	}

	if o.auth.Scope.ProjectName != "" && o.auth.Scope.ProjectID == "" && o.auth.Scope.DomainName == "" && o.auth.DomainID == "" {
		return errors.Errorf("You need to provide --project-domain-name/--project-domain-id or OS_PROJECT_DOMAIN_NAME/OS_PROJECT_DOMAIN_ID")
	}

	return nil
}

func (o *CredentialsOptions) Complete(args []string) error {
	var err error

	if err := o.setupKubeconfig(); err != nil {
		glog.V(2).Infof("%+v", err)
		return errors.Errorf("No existing kubeconfig was found")
	}

	if err := o.setupOpenstackClients(); err != nil {
		glog.V(2).Infof("%+v", err)
		return errors.Errorf("Openstack clients couldn't be created")
	}

	fmt.Printf("Authenticating %v/%v at %v/%v\n", o.auth.DomainName, o.auth.Username, o.auth.Scope.DomainName, o.auth.Scope.ProjectName)
	if err := o.authenticate(); err != nil {
		glog.V(2).Infof("%#v", err)
		return errors.Errorf("Authentication failed")
	}

	if o.url == "" {
		fmt.Println("Auto-Detectng Kubernikus Host...")
		if o.url, err = o.autoDetectKubernikusHost(); err != nil {
			glog.V(2).Infof("%+v", err)
			return errors.Errorf("You need to provide --url. Auto-Detection failed")
		}
	}

	url, err := url.Parse(o.url)
	if err != nil {
		glog.V(2).Infof("%#v", err)
		return errors.Errorf("The URL for the Kubernikus API is not parsable")
	}

	transport := kubernikus.DefaultTransportConfig().
		WithSchemes([]string{url.Scheme}).
		WithHost(url.Hostname()).
		WithBasePath(url.EscapedPath())
	o.kubernikus = kubernikus.NewHTTPClientWithConfig(nil, transport)

	if len(args) == 1 {
		o.name = args[0]
	}

	if o.name == "" {
		fmt.Println("Auto-Detecting Kubernikus Cluster...")
		if o.name, err = o.autoDetectClusterName(); err != nil {
			glog.V(2).Infof("%+v", err)
			return errors.Errorf("You need to provide --host. Auto-Detection failed")
		}
	}

	return nil
}

func (o *CredentialsOptions) Run(c *cobra.Command) error {
	fmt.Printf("Fetching credentials for %v/%v/%v from %v\n", o.auth.Scope.DomainName, o.auth.Scope.ProjectName, o.name, o.url)
	kubeconfig, err := o.fetchCredentials()
	if err != nil {
		glog.V(2).Infof("%+v", err)
		return errors.Wrap(err, "Couldn't fetch credentials from Kubernikus API")
	}

	err = o.mergeAndPersist(kubeconfig)
	if err != nil {
		glog.V(2).Infof("%+v", err)
		return errors.Errorf("Couldn't merge existing kubeconfig with fetched credentials")
	}

	fmt.Printf("Wrote merged kubeconfig to %v\n", clientcmd.NewDefaultPathOptions().GetDefaultFilename())

	return nil
}

func (o *CredentialsOptions) setupKubeconfig() error {
	var err error

	if o.kubeconfig, err = clientcmd.NewDefaultPathOptions().GetStartingConfig(); err != nil {
		return errors.Wrap(err, "Getting starting config failed")
	}

	return nil
}

func (o *CredentialsOptions) setupOpenstackClients() error {
	var err error

	if o.provider, err = openstack.NewClient(o.auth.IdentityEndpoint); err != nil {
		return errors.Wrap(err, "Creating Gophercloud ProviderClient failed")
	}

	if o.identity, err = openstack.NewIdentityV3(o.provider, gophercloud.EndpointOpts{}); err != nil {
		return errors.Wrap(err, "Creating Identity ServiceClient failed")
	}

	return nil
}

func (o *CredentialsOptions) authenticate() error {
	if err := openstack.AuthenticateV3(o.provider, o.auth, gophercloud.EndpointOpts{}); err != nil {
		return errors.Wrapf(err, "Couldn't authenticate using %#v", o.auth)
	}

	return nil
}

func (o *CredentialsOptions) autoDetectKubernikusHost() (string, error) {
	catalog, err := tokens.Create(o.identity, o.auth).ExtractServiceCatalog()
	if err != nil {
		return "", errors.Wrap(err, "Couldn't fetch service catalog")
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

	return "", errors.Errorf("No public Kubernikus service found in the service catalog")
}

func (o *CredentialsOptions) fetchCredentials() (string, error) {
	ok, err := o.kubernikus.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(o.name),
		runtime.ClientAuthInfoWriterFunc(
			func(req runtime.ClientRequest, reg strfmt.Registry) error {
				req.SetHeaderParam("X-AUTH-TOKEN", o.provider.TokenID)
				return nil
			}))

	switch err.(type) {
	case *operations.GetClusterCredentialsDefault:
		result := err.(*operations.GetClusterCredentialsDefault)
		if result.Code() == 404 {
			return "", errors.Errorf("Cluster %v not found", o.name)
		}
		return "", errors.Errorf(*result.Payload.Message)
	case error:
		return "", errors.Wrapf(err, "A generic error occured")
	}

	return ok.Payload.Kubeconfig, nil
}

func (o *CredentialsOptions) autoDetectClusterName() (string, error) {
	ok, err := o.kubernikus.Operations.ListClusters(
		operations.NewListClustersParams(),
		runtime.ClientAuthInfoWriterFunc(
			func(req runtime.ClientRequest, reg strfmt.Registry) error {
				req.SetHeaderParam("X-AUTH-TOKEN", o.provider.TokenID)
				return nil
			}))

	switch err.(type) {
	case *operations.ListClustersDefault:
		result := err.(*operations.ListClustersDefault)
		return "", errors.Errorf(*result.Payload.Message)
	case error:
		return "", errors.Wrapf(err, "Listing clusters failed")
	}

	if err != nil {
		return "", errors.Wrap(err, "Couldn't fetch kluster list from Kubernikus API")
	}

	if len(ok.Payload) == 0 {
		return "", errors.Errorf("There's no cluster in this project")
	}

	if len(ok.Payload) > 1 {
		return "", errors.Errorf("There's more than one cluster in this project. Please specify --name to select a cluster...")
	}

	return *ok.Payload[0].Name, nil
}

func (o *CredentialsOptions) mergeAndPersist(rawConfig string) error {
	config, err := clientcmd.Load([]byte(rawConfig))
	if err != nil {
		return errors.Wrapf(err, "Couldn't load kubernikus kubeconfig: %v", rawConfig)
	}

	if err := mergo.MergeWithOverwrite(o.kubeconfig, config); err != nil {
		return errors.Wrap(err, "Couldn't merge kubeconfigs")
	}

	defaultPathOptions := clientcmd.NewDefaultPathOptions()
	if err = clientcmd.ModifyConfig(defaultPathOptions, *o.kubeconfig, false); err != nil {
		return errors.Wrapf(err, "Couldn't merge Kubernikus config with kubeconfig at %v:", defaultPathOptions.GetDefaultFilename())
	}

	return nil
}
