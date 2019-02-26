package auth

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/howeyc/gopass"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	keyring "github.com/zalando/go-keyring"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
)

type InitOptions struct {
	_url string

	url            *url.URL
	name           string
	kubeconfigPath string

	openstack  *common.OpenstackClient
	kubernikus *common.KubernikusClient
}

func NewInitCommand() *cobra.Command {
	o := &InitOptions{
		name:      os.Getenv("KUBERNIKUS_NAME"),
		_url:      os.Getenv("KUBERNIKUS_URL"),
		openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:   "init",
		Short: "Prepares kubeconfig with Kubernikus credentials",
		Run: func(c *cobra.Command, args []string) {
			common.CheckError(o.Validate(c, args))
			common.CheckError(o.Complete(args))
			common.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

func (o *InitOptions) BindFlags(flags *pflag.FlagSet) {
	o.openstack.BindFlags(flags)

	flags.StringVar(&o._url, "url", o._url, "URL for Kubernikus API")
	flags.StringVar(&o.name, "name", o.name, "Cluster Name")
	flags.StringVar(&o.kubeconfigPath, "kubeconfig", o.kubeconfigPath, "Overwrites kubeconfig auto-detection with explicit path")
}

func (o *InitOptions) Validate(c *cobra.Command, args []string) (err error) {
	if o._url != "" {
		if o.url, err = url.Parse(o._url); err != nil {
			return errors.Errorf("Parsing the Kubernikus URL failed")
		}
	}
	return o.openstack.Validate(c, args)
}

func (o *InitOptions) Complete(args []string) (err error) {
	if err := o.openstack.Complete(args); err != nil {
		return err
	}

	return nil
}

func (o *InitOptions) Run(c *cobra.Command) (err error) {
	storePasswordInKeyRing := false
	if o.openstack.Password == "" {
		fmt.Printf("Password: ")
		if password, err := gopass.GetPasswdMasked(); err != nil {
			return err
		} else {
			o.openstack.Password = string(password)
			storePasswordInKeyRing = true
		}
	}

	if err := o.setup(); err != nil {

		if _, ok := errors.Cause(err).(gophercloud.ErrDefault401); o.openstack.Username != "" && ok {
			fmt.Println("Deleting password from keyring")
			keyring.Delete("kubernikus", strings.ToLower(o.openstack.Username))
		}

		return err
	}

	if o.name == "" {
		if cluster, err := o.kubernikus.GetDefaultCluster(); err != nil {
			return errors.Wrapf(err, "You need to provide --name. Cluster Auto-Detection failed")
		} else {
			o.name = cluster.Name
			glog.V(2).Infof("Detected cluster name: %v", o.name)
		}
	}

	fmt.Printf("Fetching credentials for %v from %v\n", o.name, o.url)
	kubeconfig, err := o.kubernikus.GetCredentials(o.name)
	if err != nil {
		return errors.Wrap(err, "Couldn't fetch credentials from Kubernikus API")
	}

	if storePasswordInKeyRing {
		fmt.Println("Storing password in keyring")
		keyring.Set("kubernikus", strings.ToLower(o.openstack.Username), o.openstack.Password)
	}

	ktx, err := common.NewKubernikusContext(o.kubeconfigPath, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to load kubeconfig")
	}

	if err := ktx.MergeAndPersist(kubeconfig); err != nil {
		return errors.Wrapf(err, "Couldn't merge existing kubeconfig with fetched credentials")
	}

	fmt.Printf("Updated kubeconfig at %s\n", ktx.PathOptions.GetDefaultFilename())

	return nil
}

func (o *InitOptions) setup() error {
	glog.V(2).Infof(o.openstack.PrintDebugAuthInfo())
	fmt.Println(o.openstack.PrintAuthInfo())

	if err := o.openstack.Authenticate(); err != nil {
		return errors.Wrapf(err, "Authentication failed")
	}

	if o.url == nil {
		if url, err := o.openstack.DefaultKubernikusURL(); err != nil {
			return errors.Wrapf(err, "You need to provide --url. Auto-Detection failed")
		} else {
			o.url = url
			glog.V(2).Infof("Detected Kubernikus URL: %v", url)
		}
	}

	o.kubernikus = common.NewKubernikusClient(o.url, o.openstack.Provider.TokenID)
	return nil
}
