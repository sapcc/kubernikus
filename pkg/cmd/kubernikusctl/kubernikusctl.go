package kubernikusctl

import (
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
		NewVersionCommand(),
	)

	return c
}
