package common

import (
	"os"

	kitLog "github.com/go-kit/log"
	"github.com/spf13/pflag"
	"k8s.io/klog"

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
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC, "caller", log.Caller(4))
	klog.ClampLevel(klog.Level(logLevel))
	klog.SetLogger(logger)
}
