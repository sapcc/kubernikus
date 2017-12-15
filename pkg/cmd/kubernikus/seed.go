package kubernikus

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus/seed"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewSeedCommand() *cobra.Command {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = logutil.NewTrailingNilFilter(logger)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", Caller(3))

	c := &cobra.Command{
		Use:   "seed",
		Short: "Seed stuff",
	}

	c.AddCommand(
		seed.NewKubeDNSCommand(logger),
	)

	return c
}
