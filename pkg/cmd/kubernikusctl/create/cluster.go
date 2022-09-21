package create

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/cmd"
)

func (o *CreateOptions) NewClusterCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "cluster",
		Short:   "Creates a cluster defined in a spec expected at stdin",
		Aliases: []string{"clusters"},
		PreRun:  o.clusterPreRun,
		Run:     o.clusterRun,
	}
	c.PersistentFlags().StringVarP(&o.ReadFile, "file", "f", "", "File to read spec from")
	return c
}

func (o *CreateOptions) clusterPreRun(c *cobra.Command, args []string) {
	cmd.CheckError(validateClusterCommandArgs(args))
	cmd.CheckError(o.SetupKubernikusClient())
}

func (o *CreateOptions) clusterRun(c *cobra.Command, args []string) {
	var raw []byte
	var err error
	klog.V(2).Infof("ReadFile: %v", o.ReadFile)
	if o.ReadFile != "" {
		raw, err = os.ReadFile(o.ReadFile)
		if err != nil {
			klog.V(2).Infof("error reading spec file: %v", err)
			cmd.CheckError(errors.Wrap(err, "Error reading from spec file"))
		}
	} else {
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			klog.V(2).Infof("error reading from stdin: %v", err)
			cmd.CheckError(errors.Wrap(err, "Error reading from Stdin"))
		}
	}
	klog.V(2).Infof("Raw read: \n%v", string(raw))
	var cluster models.Kluster
	cmd.CheckError(cluster.UnmarshalBinary(raw))
	klog.V(2).Infof("cluster: %v", cluster)
	cmd.CheckError(o.Kubernikus.CreateCluster(&cluster))
	fmt.Printf("Cluster %v created.", cluster.Name)
}

func validateClusterCommandArgs(args []string) error {
	if len(args) != 0 {
		return errors.Errorf("Unexpected Argument to cluster: %v", args)
	}
	return nil
}
