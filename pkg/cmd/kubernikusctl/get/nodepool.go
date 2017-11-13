package get

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (o *GetOptions) NewNodePoolCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "nodepool [cluster] [name]",
		Short:   "Gets info about nodepools of a cluster",
		Long:    `Retrieves information about a nodepool or all nodepools of a specific cluster.`,
		Aliases: []string{"nodepools", "np"},
		PreRun:  o.nodePoolPreRun,
		Run:     o.nodePoolRun,
	}
	return c
}

func (o *GetOptions) nodePoolPreRun(c *cobra.Command, args []string) {
	cmd.CheckError(validateNodePoolCommandArgs(args))
	cmd.CheckError(o.SetupKubernikusClient())
}

func (o *GetOptions) nodePoolRun(c *cobra.Command, args []string) {
	glog.V(2).Infof("Run args: %v", args)
	if len(args) == 1 {
		cmd.CheckError(o.nodePoolList(args[0]))
	} else {
		cmd.CheckError(o.nodePoolShow(args[0], args[1]))
	}
}

func (o *GetOptions) nodePoolList(cluster string) error {
	nodePools, err := o.Kubernikus.ListNodePools(cluster)
	if err != nil {
		glog.V(2).Infof("Error listing nodepools: %v", err)
		return errors.Wrap(err, "Error listing nodepools")
	}
	printme := make([]printers.Printable, len(nodePools))
	for i, nodePool := range nodePools {
		tmp := nodePool
		printme[i] = tmp
	}
	return printers.PrintTable(printme)
}

func (o *GetOptions) nodePoolShow(cluster string, nodePoolName string) error {
	nodePool, err := o.Kubernikus.ShowNodePool(cluster, nodePoolName)
	if err != nil {
		glog.V(2).Infof("Error getting nodepool %v from cluster %v: %v", nodePoolName, cluster, err)
		return errors.Wrap(err, "Error getting nodepool")
	}
	return nodePool.Print(printers.Human, printers.PrintOptions{})
}

func validateNodePoolCommandArgs(args []string) error {
	if len(args) > 2 {
		return errors.Errorf("Surplus arguments to nodepool, %v", args)
	}
	if len(args) < 1 {
		return errors.Errorf("No clustername given, %v", args)
	}
	return nil
}
