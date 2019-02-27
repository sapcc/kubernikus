package common

import (
	"os"

	kitLog "github.com/go-kit/kit/log"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/util/log"
)

var (
	logLevel int
)

func BindLogFlags(flags *pflag.FlagSet) {
	flags.IntVarP(&logLevel, "debug", "v", 0, "")
}

func SetupLogger() {
	logger := kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stderr))
	logger = log.NewTrailingNilFilter(logger)
	//logger = log.NewLevelFilter(logLevel, logger)
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC, "caller", log.Caller(4))
	glog.SetLogger(logger, int32(logLevel))

}
