package log

import (
	"fmt"

	kitlog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

type errorOrigin struct {
	next kitlog.Logger
}

// NewTrailingNilFilter removes key values pairs at the end with a nil value
// This mainly for getting rid of  err=null trailers
func NewErrorOrigin(logger kitlog.Logger) kitlog.Logger {
	return &errorOrigin{next: logger}
}

func (l errorOrigin) Log(keyvals ...interface{}) error {
	for i := len(keyvals) - 2; i >= 0; i -= 2 {
		if err, ok := keyvals[i+1].(error); ok {
			if st := originalStackTrace(err); st != nil && len(st) > 0 {
				keyvals = append(keyvals, "origin", fmt.Sprintf("%v", st[0]))
				break
			}
		}
	}
	return l.next.Log(keyvals...)
}

func originalStackTrace(err error) (st errors.StackTrace) {
	type causer interface {
		Cause() error
	}
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	for err != nil {
		if sTracer, ok := err.(stackTracer); ok {
			st = sTracer.StackTrace()
		}
		cause, ok := err.(causer)

		if !ok {
			break
		}
		err = cause.Cause()
	}
	return st
}
