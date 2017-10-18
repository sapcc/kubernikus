package kubernikusctl

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	kubernikus "github.com/sapcc/kubernikus/pkg/client/kubernikus_generated"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus_generated/operations"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/spf13/cobra"
	"net/url"
	//	"github.com/spf13/pflag"
	// "github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

type ClusterOptions struct {
	auth *CredentialsOptions
}

func (o *CredentialsOptions) setupKubernikusClient() error {
	var err error
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

	return nil
}

func NewClusterCommand() *cobra.Command {
	o := ClusterOptions{
		auth: NewCredentialsOptions(),
	}

	c := &cobra.Command{
		Use:   "cluster [command]",
		Short: "Cluster related operations via CLI",
		Long:  `Typical CRUD operations on clusters via CLI`,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.auth.Validate(c, args))
			cmd.CheckError(o.auth.setupOpenstackClients())
			cmd.CheckError(o.auth.authenticate())
			cmd.CheckError(o.auth.setupKubernikusClient())
		},
		Run: func(c *cobra.Command, args []string) {
			c.Help()
		},
	}

	o.auth.BindFlags(c.PersistentFlags())

	list := NewClusterListCommand(o)
	c.AddCommand(list)
	return c
}

func NewClusterListCommand(o ClusterOptions) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "Lists available clusters via CLI",
		Run: func(c *cobra.Command, args []string) {
			o.List()
		},
	}
	return c
}

func (o *ClusterOptions) List() error {
	fmt.Println("List")
	ok, err := o.auth.kubernikus.Operations.ListClusters(
		operations.NewListClustersParams(),
		runtime.ClientAuthInfoWriterFunc(
			func(req runtime.ClientRequest, reg strfmt.Registry) error {
				req.SetHeaderParam("X-AUTH-TOKEN", o.auth.provider.TokenID)
				return nil
			},
		))
	cmd.CheckError(err)
	for _, cluster := range ok.Payload {
		fmt.Println(cluster.Name)
	}
	return nil
}
