package kubernikusctl

import (
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/get"
	"github.com/spf13/cobra"
)

func getRun(c *cobra.Command, args []string) {
	c.Help()
}

func NewGetCommand() *cobra.Command {
	o := get.GetOptions{
		Openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:              "get [object]",
		Short:            "Retrieves Information about object from the server",
		PersistentPreRun: o.PersistentPreRun,
		Run:              getRun,
	}

	o.Openstack.BindFlags(c.PersistentFlags())
	cluster := get.NewClusterCommand(o)
	c.AddCommand(cluster)
	nodePool := get.NewNodePoolCommand(o)
	c.AddCommand(nodePool)
	return c
}
