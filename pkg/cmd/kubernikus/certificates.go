package kubernikus

import (
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/certificates"
	"github.com/spf13/cobra"
)

func NewCertificatesCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "certificates",
		Short: "Debug certificates",
	}

	c.AddCommand(
		certificates.NewFilesCommand(),
		certificates.NewPlainCommand(),
		certificates.NewSignCommand(),
	)

	return c
}
