package get

import (
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type GetOptions struct {
	_url       string
	url        *url.URL
	Openstack  *common.OpenstackClient
	Kubernikus *common.KubernikusClient
}

func (o *GetOptions) BindFlags(flags *pflag.FlagSet) {
	o.Openstack.BindFlags(flags)
	flags.StringVar(&o._url, "url", o._url, "URL for Kubernikus API")
}

func (o *GetOptions) PersistentPreRun(c *cobra.Command, args []string) {
	glog.V(2).Infof("Get PPR: %v", o)
	cmd.CheckError(o.Openstack.Validate(c, args))
	cmd.CheckError(o.Openstack.Setup())
	cmd.CheckError(o.Openstack.Authenticate())
	glog.V(2).Infof("Get PPR out: %v", o)
}

func (o *GetOptions) SetupKubernikusClient() error {
	var err error
	glog.V(2).Infof("SetupKubernikusClient called with url: %v", o._url)
	if o._url == "" {
		fmt.Println("Auto-Detecting Kubernikus Host ...")
		if o.url, err = o.Openstack.DefaultKubernikusURL(); err != nil {
			glog.V(2).Infof("Error detecting kubernikust host: %+v", err)
			return errors.Errorf("You need to provide --url. Auto-Detection failed.")
		}
	} else {
		o.url, err = url.Parse(o._url)
		if err != nil {
			glog.V(2).Infof("Error parsing url: %v", o._url)
			return errors.Wrap(err, "Error parsing url")
		}
	}
	glog.V(2).Infof("Setting up kubernikus client at %v.", o.url)
	o.Kubernikus = common.NewKubernikusClient(o.url, o.Openstack.Provider.TokenID)
	return nil
}
