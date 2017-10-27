package kubernikusctl

import (
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/create"
	"github.com/spf13/cobra"
)

func createRun(c *cobra.Command, args []string) {
	c.Help()
}

func NewCreateCommand() *cobra.Command {
	o := create.CreateOptions{
		Openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:              "create [object]",
		Short:            "Creates an object from a given spec",
		PersistentPreRun: o.PersistentPreRun,
		Run:              createRun,
	}
	o.Openstack.BindFlags(c.PersistentFlags())
	cluster := create.NewClusterCommand(o)
	c.AddCommand(cluster)
	return c
}
