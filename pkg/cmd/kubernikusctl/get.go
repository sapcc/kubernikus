package kubernikusctl

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/get"
)

func getRun(c *cobra.Command, args []string) {
	c.Help()
}

func NewGetCommand() *cobra.Command {
	o := &get.GetOptions{
		Openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:              "get [object]",
		Short:            "Retrieves Information about object from the server",
		PersistentPreRun: o.PersistentPreRun,
		Run:              getRun,
	}

	o.BindFlags(c.PersistentFlags())
	c.AddCommand(o.NewClusterCommand())
	c.AddCommand(o.NewNodePoolCommand())
	return c
}
