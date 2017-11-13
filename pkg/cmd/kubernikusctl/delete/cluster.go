package delete

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd"
)

func (o *DeleteOptions) NewClusterCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "cluster [name]",
		Short:   "Deletes the cluster with the given name",
		Aliases: []string{"clusters"},
		PreRun:  o.clusterPreRun,
		Run:     o.clusterRun,
	}

	return c
}

func (o *DeleteOptions) clusterPreRun(c *cobra.Command, args []string) {
	cmd.CheckError(validateClusterCommandArgs(args))
	cmd.CheckError(o.SetupKubernikusClient())
}

func (o *DeleteOptions) clusterRun(c *cobra.Command, args []string) {
	cmd.CheckError(o.Kubernikus.DeleteCluster(args[0]))
	fmt.Printf("Cluster %v scheduled for deletion.", args[0])
}

func validateClusterCommandArgs(args []string) error {
	if len(args) > 1 {
		return errors.Errorf("Surplus arguments to cluster delete.")
	}
	if len(args) < 1 {
		return errors.Errorf("Please supply the name of the cluster to be deleted.")
	}
	return nil
}
