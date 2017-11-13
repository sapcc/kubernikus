package create

import (
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
)

type CreateOptions struct {
	_url       string
	url        *url.URL
	Openstack  *common.OpenstackClient
	Kubernikus *common.KubernikusClient
	ReadFile   string
}

func (o *CreateOptions) PersistentPreRun(c *cobra.Command, args []string) {
	cmd.CheckError(o.Openstack.Validate(c, args))
	cmd.CheckError(o.Openstack.Setup())
	cmd.CheckError(o.Openstack.Authenticate())
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	o.Openstack.BindFlags(flags)

	flags.StringVar(&o._url, "url", o._url, "URL for Kubernikus API")
}

func (o *CreateOptions) SetupKubernikusClient() error {
	var err error
	if o._url == "" {
		fmt.Println("Auto-Detecting Kubernikus Host ...")
		if o.url, err = o.Openstack.DefaultKubernikusURL(); err != nil {
			glog.V(2).Infof("Error detecting kubernikus host: %+v", err)
			return errors.Errorf("You need to provide --url. Auto-Detection failed.")
		}
	}
	glog.V(2).Infof("Setting up kubernikus client at %v.", o.url)
	o.Kubernikus = common.NewKubernikusClient(o.url, o.Openstack.Provider.TokenID)
	return nil
}
