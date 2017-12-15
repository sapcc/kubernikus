package wormhole

import (
	"flag"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"

	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewCommand(name string) *cobra.Command {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = logutil.NewTrailingNilFilter(logger)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", kubernikus.Caller(3))

	c := &cobra.Command{
		Use:   name,
		Short: "Wormhole as a Service",
		Long:  `Creates node-aware tunnelt connections between API server and Nodes`,
	}

	c.AddCommand(
		NewServerCommand(logger),
		NewClientCommand(logger),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
