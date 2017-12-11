package kubernikusctl

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/create"
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
	o.BindFlags(c.PersistentFlags())
	c.AddCommand(o.NewClusterCommand())
	return c
}
