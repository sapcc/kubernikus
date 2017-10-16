package kubernikus

import (
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/seed"
	"github.com/spf13/cobra"
)

func NewSeedCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "seed",
		Short: "Seed stuff",
	}

	c.AddCommand(
		seed.NewKubeDNSCommand(),
	)

	return c
}
