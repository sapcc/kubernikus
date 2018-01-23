package log

import (
	"fmt"
	"os"
	"strconv"

	kitLog "github.com/go-kit/kit/log"
	"github.com/go-stack/stack"
	"github.com/spf13/pflag"
)

func NewLogger(flags *pflag.FlagSet) kitLog.Logger {
	//for now we piggyback on the --v flag defined by glog.
	level := 0
	if v := flags.Lookup("v"); v != nil {
		level, _ = strconv.Atoi(v.Value.String())
	}

	var logger kitLog.Logger
	logger = kitLog.NewLogfmtLogger(kitLog.NewSyncWriter(os.Stderr))
	logger = NewTrailingNilFilter(logger)
	logger = NewLevelFilter(level, logger)
	logger = kitLog.With(logger, "ts", kitLog.DefaultTimestampUTC, "caller", Caller(3))

	return logger

}

func Caller(depth int) kitLog.Valuer {
	return func() interface{} { return fmt.Sprintf("%+v", stack.Caller(depth)) }
}
