package guttle

import (
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
)

type StdlibAdapter struct {
	Logger log.Logger
}

const (
	logRegexpDate = `(?P<date>[0-9]{4}/[0-9]{2}/[0-9]{2})?[ ]?`
	logRegexpTime = `(?P<time>[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?)?[ ]?`
	logRegexpFile = `(?P<file>.+?:[0-9]+)?`
	logRegexpMsg  = `(: )?(?P<msg>.*)`
)

var (
	logRegexp = regexp.MustCompile(logRegexpDate + logRegexpTime + logRegexpFile + logRegexpMsg)
)

func (a StdlibAdapter) Write(p []byte) (int, error) {
	m := logRegexp.FindSubmatch(p)
	var msg string
	if m != nil {
		msg = string(m[6])
	} else {
		msg = strings.TrimSpace(string(p))
	}

	if err := a.Logger.Log("caller", log.Caller(5)(), "msg", msg); err != nil {
		return 0, err
	}
	return len(p), nil
}
