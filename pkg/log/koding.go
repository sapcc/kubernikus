package log

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/koding/logging"
)

type KodingToGlogAdapter struct {
	prefixes []interface{}
}

func (l KodingToGlogAdapter) SetLevel(_ logging.Level) {
	//Ignore
}

func (l KodingToGlogAdapter) SetHandler(_ logging.Handler) {
	//Ignore
}

func (l KodingToGlogAdapter) SetCallDepth(int) {
	//Ignore
}

func (l KodingToGlogAdapter) New(prefixes ...interface{}) logging.Logger {
	return &KodingToGlogAdapter{prefixes: append(l.prefixes, prefixes)}
}

func (l KodingToGlogAdapter) Fatal(format string, args ...interface{}) {
	glog.ExitDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Panic(format string, args ...interface{}) {
	glog.FatalDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Critical(format string, args ...interface{}) {
	glog.ErrorDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Error(format string, args ...interface{}) {
	glog.ErrorDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Warning(format string, args ...interface{}) {
	glog.WarningDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Notice(format string, args ...interface{}) {
	glog.InfoDepth(2, fmt.Sprintf(format, args...))
}
func (l KodingToGlogAdapter) Info(format string, args ...interface{}) {
	glog.InfoDepth(2, fmt.Sprintf(format, args...))
}

func (l KodingToGlogAdapter) Debug(format string, args ...interface{}) {
	if glog.V(2) {
		glog.InfoDepth(2, fmt.Sprintf(format, args...))
	}
}
