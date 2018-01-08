package kubernikusctl

import (
	"flag"

	"github.com/spf13/cobra"
)

func NewCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: "Kubernikus Kubectl Plugin",
		Long:  "Plugin that extends kubectl with Kubernikus convenience features",
	}

	c.AddCommand(
		NewAuthCommand(),
		NewGetCommand(),
		NewCreateCommand(),
		NewDeleteCommand(),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
