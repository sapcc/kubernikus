package kubernikusctl

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/version"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version information for the kubernikusctl binary",
		Run: func(c *cobra.Command, _ []string) {
			fmt.Printf("%s+%s\n", version.VERSION, version.GitCommit)

		},
	}
}
