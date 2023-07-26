package drain

import (
	"github.com/go-kit/log"
)

type LogWriter struct {
	Logger log.Logger
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	w.Logger.Log(
		"msg", string(p),
		"err", err,
	)
	return len(p), nil
}
