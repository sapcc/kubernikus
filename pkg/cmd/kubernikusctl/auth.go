package kubernikusctl

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl/auth"
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
