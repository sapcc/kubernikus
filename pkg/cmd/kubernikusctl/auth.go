package kubernikusctl

import (
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/auth"
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Authentication Commands",
	}

	c.AddCommand(
		auth.NewInitCommand(),
		auth.NewRefreshCommand(),
	)

	return c
}
