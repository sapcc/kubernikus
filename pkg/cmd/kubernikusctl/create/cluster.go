package create

import (
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/client/models"
	"github.com/sapcc/kubernikus/pkg/cmd"
)

func NewClusterCommand(o CreateOptions) *cobra.Command {
	c := &cobra.Command{
		Use:     "cluster [name]",
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
	glog.V(2).Infof("ReadFile: %v", o.ReadFile)
	if o.ReadFile != "" {
		raw, err = ioutil.ReadFile(o.ReadFile)
		if err != nil {
			glog.V(2).Infof("error reading spec file: %v", err)
			cmd.CheckError(errors.Wrap(err, "Error reading from spec file"))
		}
	} else {
		raw, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			glog.V(2).Infof("error reading from stdin: %v", err)
			cmd.CheckError(errors.Wrap(err, "Error reading from Stdin"))
		}
	}
	glog.V(2).Infof("Raw read: \n%v", string(raw))
	var cluster models.Cluster
	cmd.CheckError(cluster.UnmarshalBinary(raw))
	glog.V(2).Infof("cluster: %v", cluster)
	cmd.CheckError(o.Kubernikus.CreateCluster(&cluster))
}

func validateClusterCommandArgs(args []string) error {
	if len(args) != 0 {
		return errors.Errorf("Unexpected Argument to cluster: %v", args)
	}
	return nil
}
