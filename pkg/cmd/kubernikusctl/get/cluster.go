package get

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (o *GetOptions) NewClusterCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "cluster [name]",
		Short:   "Gets info about clusters",
		Long:    `Retrieves information regarding a cluster or all clusters from the server.`,
		Aliases: []string{"clusters"},
		PreRun:  o.clusterPreRun,
		Run:     o.clusterRun,
	}
	return c
}

func (o *GetOptions) clusterPreRun(c *cobra.Command, args []string) {
	klog.V(2).Infof("Get Cluster PR: %v", o)
	cmd.CheckError(validateClusterCommandArgs(args))
	cmd.CheckError(o.SetupKubernikusClient())
}

func (o *GetOptions) clusterRun(c *cobra.Command, args []string) {
	klog.V(2).Infof("Run args: %v", args)
	if len(args) == 1 {
		cmd.CheckError(o.clusterShow(args[0]))
	} else {
		cmd.CheckError(o.clusterList())
	}
}

func (o *GetOptions) clusterList() error {
	clusters, err := o.Kubernikus.ListAllClusters()
	if err != nil {
		klog.V(2).Infof("Error listing clusters: %v", err)
		return errors.Wrap(err, "Error listing clusters")
	}
	printme := make([]printers.Printable, len(clusters))
	for i, cluster := range clusters {
		tmp := cluster
		printme[i] = tmp
	}
	return printers.PrintTable(printme)
}

func (o *GetOptions) clusterShow(name string) error {
	cluster, err := o.Kubernikus.ShowCluster(name)
	if err != nil {
		klog.V(2).Infof("Error getting cluster %v: %v", name, err)
		return errors.Wrap(err, "Error getting cluster")
	}
	return cluster.Print(printers.Human, printers.PrintOptions{})
}

func validateClusterCommandArgs(args []string) error {
	if len(args) > 1 {
		return errors.Errorf("Surplus arguments to cluster, %v", args)
	}
	return nil
}
