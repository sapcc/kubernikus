package helm

import (
	"errors"
	"fmt"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func NewCommand() *cobra.Command {
	o := NewHelmOptions()

	c := &cobra.Command{
		Use:   "helm NAME",
		Short: "Print Helm values",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	return c
}

type HelmOptions struct {
	Name string
}

func NewHelmOptions() *HelmOptions {
	return &HelmOptions{}
}

func (o *HelmOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("you must specify the cluster's name")
	}

	return nil
}

func (o *HelmOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *HelmOptions) Run(c *cobra.Command) error {
	cluster, err := ground.NewCluster(o.Name, "localdomain")
	if err != nil {
		return err
	}

	result, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	fmt.Println(string(result))

	return nil
}
