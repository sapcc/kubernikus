package kubernikus

import (
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/certificates"
)

func NewCertificatesCommand() *cobra.Command {

	c := &cobra.Command{
		Use:   "certificates",
		Short: "Debug certificates",
	}

	c.AddCommand(
		certificates.NewFilesCommand(),
		certificates.NewPlainCommand(),
	)

	return c
}
