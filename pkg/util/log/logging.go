package log

import (
	"fmt"
	"os"

	kitLog "github.com/go-kit/log"
	"github.com/go-stack/stack"
	"k8s.io/klog"
)

func NewLogger(level int) kitLog.Logger {
	var logger kitLog.Logger

	logger = kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stderr))
	logger = NewTrailingNilFilter(logger)
	logger = NewLevelFilter(level, logger)
	logger = NewErrorOrigin(logger)
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC)
	//pass go-kit logger to klog replacment simonpasquier/klog-gokit
	klog.SetLogger(kitLog.With(logger, "caller", Caller(4)))
	klog.ClampLevel(klog.Level(level))

	return kitLog.With(logger, "caller", Caller(3))

}

func Caller(depth int) kitLog.Valuer {
	return func() interface{} { return fmt.Sprintf("%+v", stack.Caller(depth)) }
}
