package get

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/sapcc/kubernikus/pkg/cmd"
)

func (o *GetOptions) NewClusterValuesCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "values [cluster fqdn]",
		Short:  "Gets helm values for a cluster",
		PreRun: o.valuesPreRun,
		Run:    o.valuesRun,
	}
	return c
}

func (o *GetOptions) valuesPreRun(c *cobra.Command, args []string) {
	klog.V(2).Infof("Get Cluster PR: %v", o)
	cmd.CheckError(validateClusterValuesCommandArgs(args))
	cmd.CheckError(o.SetupKubernikusClient())
}

func (o *GetOptions) valuesRun(c *cobra.Command, args []string) {
	klog.V(2).Infof("Run args: %v", args)

	idx := strings.LastIndex(args[0], "-")
	name, account := args[0][:idx], args[0][idx+1:]
	yamlData, err := o.Kubernikus.GetClusterValues(account, name)
	cmd.CheckError(err)
	fmt.Println(yamlData)
}

func validateClusterValuesCommandArgs(args []string) error {
	if len(args) == 0 {
		return errors.New("required cluster fqdn missing")

	}
	if len(args) > 1 {
		return errors.Errorf("Surplus arguments to cluster, %v", args)
	}
	if !strings.Contains(args[0], "-") {
		return errors.Errorf("Provided name  '%s' does not seem to be a cluster fqdn", args)
	}
	return nil
}
