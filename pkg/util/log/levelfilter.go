package log

import (
	"errors"

	kitlog "github.com/go-kit/kit/log"
)

type levelFilter struct {
	threshold int
	next      kitlog.Logger
}

var levelKey interface{} = "v"

// NewLevelFilter filters log messages based on a level key.
// It discards log messages
func NewLevelFilter(level int, logger kitlog.Logger) kitlog.Logger {
	return &levelFilter{threshold: level, next: logger}
}

func (l levelFilter) Log(keyvals ...interface{}) error {
	for i := len(keyvals) - 2; i >= 0; i -= 2 {
		if keyvals[i] == levelKey {
			var lvl int
			switch n := keyvals[i+1].(type) {
			case int:
				lvl = n
			case int32:
				lvl = int(n)
			case int64:
				lvl = int(n)
			default:
				return errors.New("Level value is not of expected type (int)")
			}
			if lvl <= l.threshold {
				return l.next.Log(keyvals...)
			}
			return nil // filter log message
		}
	}
	//ALways log lines without a level
	return l.next.Log(keyvals...)
}
