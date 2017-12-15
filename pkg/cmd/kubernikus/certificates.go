package kubernikus

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/certificates"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewCertificatesCommand() *cobra.Command {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = logutil.NewTrailingNilFilter(logger)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", Caller(3))

	c := &cobra.Command{
		Use:   "certificates",
		Short: "Debug certificates",
	}

	c.AddCommand(
		certificates.NewFilesCommand(),
		certificates.NewPlainCommand(),
		certificates.NewSignCommand(logger),
	)

	return c
}
