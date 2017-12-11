package kubernikus

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/seed"
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
