package auth

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/howeyc/gopass"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type RefreshOptions struct {
	kubeconfigPath string
	context        string

	url *url.URL

	openstack  *common.OpenstackClient
	kubernikus *common.KubernikusClient

	kubeconfig *clientcmdapi.Config
}

func NewRefreshCommand() *cobra.Command {
	o := &RefreshOptions{
		openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:   "refresh",
		Short: "Refreshes already existing credentials in kubeconfig",
		Run: func(c *cobra.Command, args []string) {
			common.CheckError(o.Validate(c, args))
			common.CheckError(o.Complete(args))
			common.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())
	return c
}

func (o *RefreshOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.openstack.Password, "password", o.openstack.Password, "User password [OS_PASSWORD]")
	flags.StringVar(&o.kubeconfigPath, "kubeconfig", o.kubeconfigPath, "Overwrites kubeconfig auto-detection with explicit path")
	flags.StringVar(&o.context, "context", o.context, "Overwrites current-context in kubeconfig")
}

func (o *RefreshOptions) Validate(c *cobra.Command, args []string) (err error) {
	return nil
}

func (o *RefreshOptions) Complete(args []string) (err error) {
	if err := o.openstack.Complete(args); err != nil {
		return err
	}

	if o.kubeconfigPath != "" {
		if err := o.loadKubeconfig(); err != nil {
			return errors.Wrapf(err, "Loading the specified kubeconfig failed")
		}
	} else {
		o.kubeconfig, err = clientcmd.NewDefaultPathOptions().GetStartingConfig()
		if err != nil {
			return errors.Wrapf(err, "Loading the default kubeconfig failed")
		}
	}

	if o.context == "" && o.kubeconfig.CurrentContext != "" {
		o.context = o.kubeconfig.CurrentContext
	}

	if o.kubeconfig.Contexts[o.context] == nil {
		return errors.Errorf("The context you provided does not exist")
	}

	glog.V(2).Infof("Using context %v", o.context)
	if isKubernikusContext, err := o.isKubernikusContext(); err != nil {
		glog.V(2).Infof("Not a valid Kubernikus context: %v", err)
		return nil
	} else {
		if !isKubernikusContext {
			glog.V(2).Infof("Not a valid Kubernikus context")
			return nil
		}
	}

	if identityEndpoint, err := o.autoDetectAuthURL(); err != nil {
		errors.Wrap(err, "Auto-Detection of auth-url caused an error")
	} else {
		glog.V(2).Infof("Detected auth-url: %v", identityEndpoint)
		o.openstack.IdentityEndpoint = identityEndpoint
	}

	if projectID, err := o.autoDetectProjectID(); err != nil {
		errors.Wrap(err, "Auto-Detection of project scope caused an error")
	} else {
		glog.V(2).Infof("Detected authentication scope for project-id: %v", projectID)
		o.openstack.Scope.ProjectID = projectID
	}

	if kurl, err := o.autoDetectKubernikusURL(); err != nil {
		errors.Wrap(err, "Auto-Detection of Kubernikus URL caused an error")
	} else {
		glog.V(2).Infof("Detected Kubernikus URL: %v", kurl)
		_url, err := url.Parse(kurl)
		if err != nil {
			return errors.Wrap(err, "Couldn't parse Kubernikus URL. Rerun init.")
		}
		o.url = _url
	}

	if username, err := o.autoDetectUsername(); err != nil {
		errors.Wrap(err, "Auto-Detection of Username failed")
	} else {
		glog.V(2).Infof("Detected username: %v", username)
		o.openstack.Username = username
	}

	if domainName, err := o.autoDetectUserDomainName(); err != nil {
		errors.Wrap(err, "Auto-Detection of user-domain-name failed")
	} else {
		glog.V(2).Infof("Detected domain-name: %v", domainName)
		o.openstack.DomainName = domainName
	}

	if o.openstack.Password == "" {
		fmt.Printf("Password: ")
		if password, err := gopass.GetPasswdMasked(); err != nil {
			return err
		} else {
			o.openstack.Password = string(password)
		}
	}

	return nil
}

func (o *RefreshOptions) Run(c *cobra.Command) error {
	if isKubernikusContext, err := o.isKubernikusContext(); err != nil {
		return nil
	} else {
		if !isKubernikusContext {
			return nil
		}
	}

	if ok, err := o.isCertificateValid(); err != nil {
		return errors.Wrap(err, "Verification of certifcates failed.")
	} else {
		if ok {
			glog.V(2).Infof("Certificates are good. Doing nothing.")
			return nil
		}
	}

	if err := o.setupClients(); err != nil {
		return err
	}

	fmt.Printf("Fetching credentials for %v from %v\n", o.context, o.url)
	kubeconfig, err := o.kubernikus.GetCredentials(o.context)
	if err != nil {
		return errors.Wrap(err, "Couldn't fetch credentials from Kubernikus API")
	}

	err = o.mergeAndPersist(kubeconfig)
	if err != nil {
		return errors.Wrapf(err, "Couldn't merge existing kubeconfig with fetched credentials")
	}

	fmt.Printf("Wrote merged kubeconfig to %v\n", clientcmd.NewDefaultPathOptions().GetDefaultFilename())

	return nil
}

func (o *RefreshOptions) setupClients() error {
	if err := o.openstack.Setup(); err != nil {
		return err
	}

	glog.V(2).Infof(o.openstack.PrintDebugAuthInfo())
	fmt.Println(o.openstack.PrintAuthInfo())

	if err := o.openstack.Authenticate(); err != nil {
		return err
	}

	o.kubernikus = common.NewKubernikusClient(o.url, o.openstack.Provider.TokenID)

	return nil
}

func (o *RefreshOptions) loadKubeconfig() (err error) {
	if o.kubeconfig, err = clientcmd.LoadFromFile(o.kubeconfigPath); err != nil {
		return errors.Wrapf(err, "Failed to load kubeconfig from %v", o.kubeconfigPath)
	}
	return nil
}

func (o *RefreshOptions) isKubernikusContext() (bool, error) {
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

func (o *RefreshOptions) autoDetectKubernikusCAMetadata(index int) (string, error) {
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

func (o *RefreshOptions) autoDetectKubernikusClientMetadata() (string, string, error) {
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

func (o *RefreshOptions) autoDetectAuthURL() (string, error) {
	return o.autoDetectKubernikusCAMetadata(0)
}

func (o *RefreshOptions) autoDetectProjectID() (string, error) {
	return o.autoDetectKubernikusCAMetadata(1)
}

func (o *RefreshOptions) autoDetectKubernikusURL() (string, error) {
	cert, err := o.getCACertifciate()
	if err != nil {
		return "", err
	}

	if len(cert.Issuer.Locality) == 0 {
		return "", errors.Errorf("CA certificate didn't contain Kubernikus metadata")
	}
	return cert.Issuer.Locality[0], nil
}

func (o *RefreshOptions) autoDetectUsername() (string, error) {
	user, _, err := o.autoDetectKubernikusClientMetadata()
	if err != nil {
		return "", err
	}
	return user, nil
}

func (o *RefreshOptions) autoDetectUserDomainName() (string, error) {
	_, domain, err := o.autoDetectKubernikusClientMetadata()
	if err != nil {
		return "", err
	}
	return domain, nil
}

func (o *RefreshOptions) getRawClientCertificate() ([]byte, error) {
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

func (o *RefreshOptions) getRawCACertificate() ([]byte, error) {
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

func (o *RefreshOptions) getCACertifciate() (*x509.Certificate, error) {
	data, err := o.getRawCACertificate()
	if err != nil {
		return nil, err
	}
	return parseRawPEM(data)
}

func (o *RefreshOptions) getClientCertificate() (*x509.Certificate, error) {
	data, err := o.getRawClientCertificate()
	if err != nil {
		return nil, err
	}
	return parseRawPEM(data)
}

func (o *RefreshOptions) isCertificateValid() (bool, error) {
	cert, err := o.getClientCertificate()
	if err != nil {
		return false, err
	}

	if time.Now().After(cert.NotAfter) || time.Now().Before(cert.NotBefore) {
		return false, nil
	}

	return true, nil
}

func (o *RefreshOptions) mergeAndPersist(rawConfig string) error {
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
