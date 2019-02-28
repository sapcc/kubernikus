package log

import (
	"fmt"
	"os"

	kitLog "github.com/go-kit/kit/log"
	"github.com/go-stack/stack"
	"github.com/golang/glog"
)

func NewLogger(level int) kitLog.Logger {
	var logger kitLog.Logger

	logger = kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stderr))
	logger = NewTrailingNilFilter(logger)
	logger = NewLevelFilter(level, logger)
	logger = NewErrorOrigin(logger)
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC, "caller", Caller(3))
	glog.SetLogger(logger, int32(level))

	return logger

}

func Caller(depth int) kitLog.Valuer {
	return func() interface{} { return fmt.Sprintf("%+v", stack.Caller(depth)) }
}
