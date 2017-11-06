package kubernikusctl

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/common"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/delete"
)

func deleteRun(c *cobra.Command, args []string) {
	c.Help()
}

func NewDeleteCommand() *cobra.Command {
	o := delete.DeleteOptions{
		Openstack: common.NewOpenstackClient(),
	}

	c := &cobra.Command{
		Use:              "delete [object]",
		Short:            "Deletes an object",
		PersistentPreRun: o.PersistentPreRun,
		Run:              deleteRun,
	}
	o.Openstack.BindFlags(c.PersistentFlags())
	cluster := delete.NewClusterCommand(o)
	c.AddCommand(cluster)
	return c
}
