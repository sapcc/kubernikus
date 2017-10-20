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
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
	"github.com/spf13/cobra"
	"net/url"
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
	show := NewClusterShowCommand(o)
	c.AddCommand(list)
	c.AddCommand(show)
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

func NewClusterShowCommand(o ClusterOptions) *cobra.Command {
	c := &cobra.Command{
		Use:   "show [name]",
		Short: "Displays information on a specific cluster",
		PreRun: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.ValidateShowArgs(c, args))
		},
		Run: func(c *cobra.Command, args []string) {
			o.Show(args[0])
		},
	}
	return c
}

func (o *ClusterOptions) ValidateShowArgs(c *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.Errorf("You need to provide a clustername to show")
	}
	if len(args) > 1 {
		return errors.Errorf("Surplus arguments to show %v", args)
	}
	return nil
}

func (o *ClusterOptions) Show(name string) error {
	params := operations.NewShowClusterParams()
	params.Name = name
	ok, err := o.auth.kubernikus.Operations.ShowCluster(
		params,
		runtime.ClientAuthInfoWriterFunc(
			func(req runtime.ClientRequest, reg strfmt.Registry) error {
				req.SetHeaderParam("X-AUTH-TOKEN", o.auth.provider.TokenID)
				return nil
			},
		))
	cmd.CheckError(err)
	ok.Payload.Print(printers.Human, printers.PrintOptions{})
	return nil
}

func (o *ClusterOptions) List() error {
	ok, err := o.auth.kubernikus.Operations.ListClusters(
		operations.NewListClustersParams(),
		runtime.ClientAuthInfoWriterFunc(
			func(req runtime.ClientRequest, reg strfmt.Registry) error {
				req.SetHeaderParam("X-AUTH-TOKEN", o.auth.provider.TokenID)
				return nil
			},
		))
	cmd.CheckError(err)
	printme := make([]printers.Printable, len(ok.Payload))
	for i, cluster := range ok.Payload {
		tmp := cluster
		printme[i] = tmp
	}
	cmd.CheckError(printers.PrintTable(printme))
	return nil
}
