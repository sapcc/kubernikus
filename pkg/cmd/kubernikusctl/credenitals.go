package kubernikusctl

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strings"

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
	"github.com/sapcc/kubernikus/pkg/util"
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
	url            string
	name           string
	kubeconfigPath string
	context        string

	kubernikus *kubernikus.Kubernikus
	auth       *tokens.AuthOptions
	provider   *gophercloud.ProviderClient
	identity   *gophercloud.ServiceClient
	kubeconfig *clientcmdapi.Config
}

func NewCredentialsOptions() *CredentialsOptions {
	o := &CredentialsOptions{
		name: os.Getenv("KUBERNIKUS_NAME"),
		url:  os.Getenv("KUBERNIKUS_URL"),
	}
	o.auth = &tokens.AuthOptions{
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
	flags.StringVar(&o.kubeconfigPath, "kubeconfig", o.kubeconfigPath, "Overwrites kubeconfig auto-detection with explicit path")
	flags.StringVar(&o.context, "context", o.context, "Overwrites current-context in kubeconfig")
}

func (o *CredentialsOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *CredentialsOptions) Complete(args []string) error {
	if len(args) == 1 {
		o.name = args[0]
	}

	return nil
}

func (o *CredentialsOptions) Run(c *cobra.Command) error {
	var err error

	// Collect Starting Kubeconfig
	// if current context is a kubernikus context
	//   if ! expired(client certificate) && ! --force
	//     return
	//   else
	//     guess auth-url/projectID/kubernikusHost from CA certificate
	//     guess username/domainName from client certificate
	//  authenticate
	//  unless --name guess kluster name
	//  unless --url auto-detect kubernikus url
	//  fetch credentials
	//  update kubeconfigs

	if o.kubeconfigPath == "" {
		o.kubeconfigPath = clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	}

	if o.kubeconfigPath != "" {
		if err := o.loadKubeconfig(); err != nil {
			glog.V(3).Infof("%v", err)
			return errors.Errorf("Loading of the specified kubeconfig failed")
		}
		glog.V(2).Infof("Loaded kubeconfig from %v", o.kubeconfigPath)
	}

	if o.context == "" && o.kubeconfig.CurrentContext != "" {
		o.context = o.kubeconfig.CurrentContext
	}

	if o.isContextValid() {
		glog.V(2).Infof("Using context %v", o.context)
		isKubernikusContext := false
		if isKubernikusContext, err = o.isKubernikusContext(); err != nil {
			glog.V(3).Infof("%v", err)
			glog.V(2).Infof("Detection of Kubernikus issued certificates failed")
		}

		if isKubernikusContext {
			ok := false
			if ok, err = o.isCertificateValid(); err != nil {
				glog.V(3).Infof("%+v", err)
			}
			if ok {
				return nil
			}

			if o.auth.IdentityEndpoint == "" {
				if identityEndpoint, err := o.autoDetectAuthURL(); err != nil {
					glog.V(3).Infof("%v", err)
					glog.V(2).Infof("Auto-Detection of auth-url caused an error")
				} else {
					o.auth.IdentityEndpoint = identityEndpoint
					glog.V(2).Infof("Detected auth-url: %v", o.auth.IdentityEndpoint)
				}
			}

			if o.auth.Scope.ProjectName == "" && o.auth.Scope.ProjectID == "" {
				if projectID, err := o.autoDetectProjectID(); err != nil {
					glog.V(3).Infof("%v", err)
					glog.V(2).Infof("Auto-Detection of project scope caused an error")
				} else {
					o.auth.Scope.ProjectID = projectID
					glog.V(2).Infof("Detected authentication scope for project-id: %v", o.auth.Scope.ProjectID)
				}
			}

			if o.url == "" {
				if url, err := o.autoDetectKubernikusURL(); err != nil {
					glog.V(3).Infof("%v", err)
					glog.V(2).Infof("Auto-Detection of Kubernikus URL caused an error")
				} else {
					o.url = url
					glog.V(2).Infof("Detected Kubernikus URL: %v", o.url)
				}
			}

			if o.auth.Username == "" {
				if username, err := o.autoDetectUsername(); err != nil {
					glog.V(3).Infof("%v", err)
					glog.V(2).Infof("Auto-Detection of Username failed")
				} else {
					o.auth.Username = username
					glog.V(2).Infof("Detected username: %v", o.auth.Username)
				}
			}

			if o.auth.DomainName == "" {
				if domainName, err := o.autoDetectUserDomainName(); err != nil {
					glog.V(3).Infof("%v", err)
					glog.V(2).Infof("Auto-Detection of user-domain-name failed")
				} else {
					o.auth.DomainName = domainName
					glog.V(2).Infof("Detected domain-name: %v", o.auth.DomainName)
				}
			}
		}
	} else {
		glog.V(2).Infof("Unkown context %v. Ignoring it.", o.context)
	}

	if o.auth.IdentityEndpoint == "" {
		return errors.Errorf("You need to provide --auth-url or OS_AUTH_URL")
	}

	if o.auth.Username == "" && o.auth.UserID == "" {
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

	if err := o.setupOpenstackClients(); err != nil {
		glog.V(3).Infof("%+v", err)
		return errors.Errorf("Openstack clients couldn't be created")
	}

	glog.V(2).Infof("Using AuthOptions:")
	glog.V(2).Infof("  IdentityEndpoint: %v", o.auth.IdentityEndpoint)
	glog.V(2).Infof("  Username:         %v", o.auth.Username)
	glog.V(2).Infof("  UserID:           %v", o.auth.UserID)
	glog.V(2).Infof("  Password:         %v", o.auth.Password)
	glog.V(2).Infof("  DomainID:         %v", o.auth.DomainID)
	glog.V(2).Infof("  DomainName:       %v", o.auth.DomainName)
	glog.V(2).Infof("  Scope:")
	glog.V(2).Infof("    ProjectID:      %v", o.auth.Scope.ProjectID)
	glog.V(2).Infof("    ProjectName:    %v", o.auth.Scope.ProjectName)
	glog.V(2).Infof("    DomainID:       %v", o.auth.Scope.DomainID)
	glog.V(2).Infof("    DomainName:     %v", o.auth.Scope.DomainName)
	fmt.Println(o.printAuthInfo())

	if err := o.authenticate(); err != nil {
		glog.V(3).Infof("%#v", err)
		return errors.Errorf("Authentication failed")
	}

	if o.url == "" {
		if url, err := o.autoDetectKubernikusURLFromServiceCatalog(); err != nil {
			glog.V(3).Infof("%v", err)
			return errors.Errorf("You need to provide --url. Auto-Detection failed")
		} else {
			o.url = url
			glog.V(2).Infof("Detected Kubernikus URL: %v", url)
		}
	}

	url, err := url.Parse(o.url)
	if err != nil {
		glog.V(3).Infof("%v", err)
		return errors.Errorf("The URL for the Kubernikus API is not parsable")
	}

	transport := kubernikus.DefaultTransportConfig().
		WithSchemes([]string{url.Scheme}).
		WithHost(url.Hostname()).
		WithBasePath(url.EscapedPath())
	o.kubernikus = kubernikus.NewHTTPClientWithConfig(nil, transport)

	if o.name == "" {
		if name, err := o.autoDetectClusterName(); err != nil {
			glog.V(3).Infof("%v", err)
			return errors.Errorf("You need to provide --host. Auto-Detection failed")
		} else {
			o.name = name
			glog.V(2).Infof("Detected cluster name: %v", name)
		}
	}

	fmt.Printf("Fetching credentials for %v from %v\n", o.name, o.url)
	kubeconfig, err := o.fetchCredentials()
	if err != nil {
		glog.V(3).Infof("%v", err)
		return errors.Wrap(err, "Couldn't fetch credentials from Kubernikus API")
	}

	err = o.mergeAndPersist(kubeconfig)
	if err != nil {
		glog.V(3).Infof("%v", err)
		return errors.Errorf("Couldn't merge existing kubeconfig with fetched credentials")
	}

	fmt.Printf("Wrote merged kubeconfig to %v\n", clientcmd.NewDefaultPathOptions().GetDefaultFilename())

	return nil
}

func (o *CredentialsOptions) loadKubeconfig() (err error) {
	if o.kubeconfig, err = clientcmd.LoadFromFile(o.kubeconfigPath); err != nil {
		return errors.Wrapf(err, "Failed to load kubeconfig from %v", o.kubeconfigPath)
	}
	return nil
}

func (o *CredentialsOptions) isContextValid() bool {
	if o.context == "" {
		return false
	}
	return o.kubeconfig.Contexts[o.context] != nil
}

func (o *CredentialsOptions) isKubernikusContext() (bool, error) {
	caCert, err := o.getCACertifciate()
	if err != nil {
		return false, err
	}

	if len(caCert.Issuer.OrganizationalUnit) < 2 {
		return false, nil
	}

	return caCert.Issuer.OrganizationalUnit[0] == util.CA_ISSUER_KUBERNIKUS_IDENTIFIER_0 &&
		caCert.Issuer.OrganizationalUnit[1] == util.CA_ISSUER_KUBERNIKUS_IDENTIFIER_1, nil
}

func (o *CredentialsOptions) autoDetectKubernikusCAMetadata(index int) (string, error) {
	cert, err := o.getCACertifciate()
	if err != nil {
		return "", err
	}
	if len(cert.Issuer.Province) < 1 {
		return "", errors.Errorf("CA certificate didn't contain Kubernikus metadata")
	}
	if index > 1 {
		return "", errors.Errorf("Invalid Metadata")
	}
	return cert.Issuer.Province[index], nil
}

func (o *CredentialsOptions) autoDetectKubernikusClientMetadata() (string, string, error) {
	cert, err := o.getClientCertificate()
	if err != nil {
		return "", "", err
	}
	if cert.Subject.CommonName == "" {
		return "", "", errors.Errorf("Client certificate didn't contain username")
	}

	parts := strings.Split(cert.Subject.CommonName, "@")
	if len(parts) != 2 {
		return "", "", errors.Errorf("Couldln't extract username/domain from client certificate %v", parts)
	}

	return parts[0], parts[1], nil
}

func (o *CredentialsOptions) autoDetectAuthURL() (string, error) {
	return o.autoDetectKubernikusCAMetadata(0)
}

func (o *CredentialsOptions) autoDetectProjectID() (string, error) {
	return o.autoDetectKubernikusCAMetadata(1)
}

func (o *CredentialsOptions) autoDetectKubernikusURL() (string, error) {
	cert, err := o.getCACertifciate()
	if err != nil {
		return "", err
	}

	if len(cert.Issuer.Locality) == 0 {
		return "", errors.Errorf("CA certificate didn't contain Kubernikus metadata")
	}
	return cert.Issuer.Locality[0], nil
}

func (o *CredentialsOptions) autoDetectUsername() (string, error) {
	user, _, err := o.autoDetectKubernikusClientMetadata()
	if err != nil {
		return "", err
	}
	return user, nil
}

func (o *CredentialsOptions) autoDetectUserDomainName() (string, error) {
	_, domain, err := o.autoDetectKubernikusClientMetadata()
	if err != nil {
		return "", err
	}
	return domain, nil
}

func (o *CredentialsOptions) getRawClientCertificate() ([]byte, error) {
	context := o.kubeconfig.Contexts[o.context]
	if context == nil {
		return nil, errors.Errorf("Couldn't find context %v", o.context)
	}

	authInfo := o.kubeconfig.AuthInfos[context.AuthInfo]
	if authInfo == nil {
		return nil, errors.Errorf("Couldn't find auth-info %v for context %v", context.AuthInfo, o.context)
	}

	cluster := o.kubeconfig.Clusters[context.Cluster]
	if cluster == nil {
		return nil, errors.Errorf("Couldn't find cluster %v", context.Cluster)
	}

	certData := authInfo.ClientCertificateData
	if certData == nil {
		return nil, errors.Errorf("Couldn't find client certificate for auth-info %v", authInfo.Username)
	}

	return certData, nil
}

func (o *CredentialsOptions) getRawCACertificate() ([]byte, error) {
	context := o.kubeconfig.Contexts[o.context]
	if context == nil {
		return nil, errors.Errorf("Couldn't find context %v", o.context)
	}

	authInfo := o.kubeconfig.AuthInfos[context.AuthInfo]
	if authInfo == nil {
		return nil, errors.Errorf("Couldn't find auth-info %v for context %v", context.AuthInfo, o.context)
	}

	cluster := o.kubeconfig.Clusters[context.Cluster]
	if cluster == nil {
		return nil, errors.Errorf("Couldn't find cluster %v", context.Cluster)
	}

	certData := cluster.CertificateAuthorityData
	if certData == nil {
		return nil, errors.Errorf("Couldn't find CA certificate for cluster %v", context.Cluster)
	}

	return certData, nil
}

func (o *CredentialsOptions) getCACertifciate() (*x509.Certificate, error) {
	data, err := o.getRawCACertificate()
	if err != nil {
		return nil, err
	}
	return parseRawPEM(data)
}

func (o *CredentialsOptions) getClientCertificate() (*x509.Certificate, error) {
	data, err := o.getRawClientCertificate()
	if err != nil {
		return nil, err
	}
	return parseRawPEM(data)
}

func (o *CredentialsOptions) isCertificateValid() (bool, error) {
	caData, err := o.getRawCACertificate()
	if err != nil {
		return false, err
	}

	cert, err := o.getClientCertificate()
	if err != nil {
		return false, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caData)

	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return false, nil
	}

	return true, nil
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

func (o *CredentialsOptions) autoDetectKubernikusURLFromServiceCatalog() (string, error) {
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

func (o *CredentialsOptions) printAuthInfo() string {
	var user, scope string

	if o.auth.UserID != "" {
		user = o.auth.UserID
	} else {
		if o.auth.DomainID != "" {
			user = fmt.Sprintf("%v/%v", o.auth.DomainID, o.auth.Username)
		} else {
			user = fmt.Sprintf("%v/%v", o.auth.DomainName, o.auth.Username)
		}
	}

	if o.auth.Scope.ProjectID != "" {
		scope = o.auth.Scope.ProjectID
	} else {
		if o.auth.Scope.DomainID != "" {
			scope = fmt.Sprintf("%v/%v", o.auth.Scope.DomainID, o.auth.Scope.ProjectName)
		} else {
			scope = fmt.Sprintf("%v/%v", o.auth.Scope.DomainName, o.auth.Scope.ProjectName)
		}
	}

	return fmt.Sprintf("Authenticating %v at %v", user, scope)
}

func parseRawPEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("Couldn't decode raw certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't parse certificate")
	}

	return cert, nil
}
