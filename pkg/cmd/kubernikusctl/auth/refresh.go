package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	keyring "github.com/zalando/go-keyring"
	"k8s.io/klog"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
)

type RefreshOptions struct {
	kubeconfigPath string
	context        string
	force          bool

	url *url.URL

	openstack  *common.OpenstackClient
	kubernikus *common.KubernikusClient
}

func NewRefreshCommand() *cobra.Command {
	o := &RefreshOptions{
		openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:   "refresh",
		Short: "Refreshes already existing credentials in kubeconfig",
		Run: func(c *cobra.Command, args []string) {
			common.SetupLogger()
			common.CheckError(o.Validate(c, args))
			common.CheckError(o.Complete(args))
			common.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())
	return c
}

func (o *RefreshOptions) BindFlags(flags *pflag.FlagSet) {
	common.BindLogFlags(flags)
	flags.StringVar(&o.openstack.Password, "password", "", "User password [OS_PASSWORD]")
	flags.StringVar(&o.openstack.TokenID, "token", "", "Token to authenticate with [OS_TOKEN]")
	flags.StringVar(&o.kubeconfigPath, "kubeconfig", o.kubeconfigPath, "Overwrites kubeconfig auto-detection with explicit path")
	flags.StringVar(&o.context, "context", o.context, "Overwrites current-context in kubeconfig")
	flags.BoolVar(&o.force, "force", o.force, "Force refresh")
}

func (o *RefreshOptions) Validate(c *cobra.Command, args []string) (err error) {
	return nil
}

func (o *RefreshOptions) Complete(args []string) (err error) {
	if err := o.openstack.Complete(args); err != nil {
		return err
	}

	return nil
}

func (o *RefreshOptions) Run(c *cobra.Command) error {

	klog.V(2).Infof("Using context %v", o.context)
	ktx, err := common.NewKubernikusContext(o.kubeconfigPath, o.context)
	if err != nil {
		return errors.Wrapf(err, "Failed to load kubeconfig")
	}
	if isKubernikusCtx, err := ktx.IsKubernikusContext(); err != nil || !isKubernikusCtx {
		klog.V(2).Infof("%s is not a valid Kubernikus context: %v", o.context, err)
		return nil
	}

	if ok, err := ktx.UserCertificateValid(); err != nil {
		return errors.Wrap(err, "Verification of certificate failed.")
	} else {
		if ok && !o.force {
			klog.V(2).Infof("Certificates are good. Doing nothing.")
			return nil
		}
	}

	authURL, err := ktx.AuthURL()
	if err != nil {
		return errors.Wrap(err, "Couldn't get AuthURL")
	}
	projectID, err := ktx.ProjectID()
	if err != nil {
		return errors.Wrap(err, "Couldn't get project ID")
	}

	klog.V(2).Infof("Detected auth-url: %v", authURL)
	o.openstack.IdentityEndpoint = authURL
	klog.V(2).Infof("Detected authentication scope for project-id: %v", projectID)
	o.openstack.Scope.ProjectID = projectID
	//Ignore conflicting values from environment
	o.openstack.Scope.ProjectName = ""
	o.openstack.Scope.DomainID = ""
	o.openstack.Scope.DomainName = ""

	kurl, err := ktx.KubernikusURL()
	if err != nil {
		return errors.Wrap(err, "Couldn't get kubernikus URL from certificate")
	}
	klog.V(2).Infof("Detected Kubernikus URL: %v", kurl)
	if o.url, err = url.Parse(kurl); err != nil {
		return errors.Wrap(err, "Couldn't parse Kubernikus URL. Rerun init.")
	}

	storePasswordInKeyRing := false
	if o.openstack.TokenID == "" {
		if o.openstack.Username, err = ktx.Username(); err != nil {
			return errors.Wrap(err, "Failed to extract username from certificate")
		}
		klog.V(2).Infof("Detected username: %v", o.openstack.Username)
		o.openstack.UserID = "" //Ignore conflicting value from env environment

		if o.openstack.DomainName, err = ktx.UserDomainname(); err != nil {
			return errors.Wrap(err, "Failed to extract user domain from certificate")
		}
		klog.V(2).Infof("Detected domain-name: %v", o.openstack.DomainName)
		o.openstack.DomainID = "" //Ignore conflicting value from environment

		if o.openstack.Password == "" {
			fmt.Printf("Password: ")
			if password, err := gopass.GetPasswdMasked(); err != nil {
				return err
			} else {
				o.openstack.Password = string(password)
				storePasswordInKeyRing = true
			}
		}
	}

	if err := o.setupClients(); err != nil {

		if o.openstack.Username != "" {
			keyring.Delete("kubernikus", strings.ToLower(o.openstack.Username))
		}

		return err
	}

	fmt.Printf("Fetching credentials for %v from %v\n", o.context, o.url)
	kubeconfig, err := o.kubernikus.GetCredentials(o.context)
	if err != nil {
		return errors.Wrap(err, "Couldn't fetch credentials from Kubernikus API")
	}

	if storePasswordInKeyRing {
		fmt.Println("Storing password in keyring")
		keyring.Set("kubernikus", strings.ToLower(o.openstack.Username), o.openstack.Password)
	}

	err = ktx.MergeAndPersist(kubeconfig)
	if err != nil {
		return errors.Wrapf(err, "Couldn't merge existing kubeconfig with fetched credentials")
	}

	fmt.Printf("Updated kubeconfig at %v\n", ktx.PathOptions.GetDefaultFilename())

	return nil
}

func (o *RefreshOptions) setupClients() error {
	if err := o.openstack.Setup(); err != nil {
		return err
	}

	klog.V(2).Infof(o.openstack.PrintDebugAuthInfo())
	fmt.Println(o.openstack.PrintAuthInfo())

	if err := o.openstack.Authenticate(); err != nil {
		return err
	}

	o.kubernikus = common.NewKubernikusClient(o.url, o.openstack.Provider.TokenID, false)

	return nil
}
