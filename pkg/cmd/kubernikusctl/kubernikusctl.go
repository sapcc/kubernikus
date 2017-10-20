package kubernikusctl

import (
	"flag"

	"github.com/spf13/cobra"
)

func NewCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: "Kubernikus Kubectl Plugin",
		Long:  "Plugin that extends kubectl with Kubernikus convinience features",
	}

	c.AddCommand(
		NewCredentialsCommand(),
		NewClusterCommand(),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
