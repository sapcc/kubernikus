package certificates

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "certificates",
		Short: "Debug certificates",
	}

	c.AddCommand(
		NewFilesCommand(),
		NewPlainCommand(),
		NewSignCommand(),
	)

	return c
}
