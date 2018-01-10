package wormhole

import (
	"flag"

	"github.com/spf13/cobra"
)

func NewCommand(name string) *cobra.Command {

	c := &cobra.Command{
		Use:   name,
		Short: "Wormhole as a Service",
		Long:  `Creates node-aware tunnelt connections between API server and Nodes`,
	}

	c.AddCommand(
		NewServerCommand(),
		NewClientCommand(),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
