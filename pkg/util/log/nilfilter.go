package log

import (
	kitlog "github.com/go-kit/kit/log"
)

type nilFilter struct {
	next kitlog.Logger
}

// NewTrailingNilFilter removes key values pairs at the end with a nil value
// This mainly for getting rid of  err=null trailers
func NewTrailingNilFilter(logger kitlog.Logger) kitlog.Logger {
	return &nilFilter{next: logger}
}

func (l nilFilter) Log(keyvals ...interface{}) error {
	for i := len(keyvals) - 1; i > 0; i -= 2 {
		if keyvals[i] != nil {
			return l.next.Log(keyvals[:i+1]...)
		}
	}
	return l.next.Log(keyvals...)
}
