package glog

import (
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
)

var logger = log.NewNopLogger()
var mu = sync.Mutex{}
var v int32
var c int32

func SetLogger(l log.Logger, level int32) {
	mu.Lock()
	logger = l
	v = level
	mu.Unlock()
}

type Level int32

type Verbose bool

func V(level Level) Verbose {
	mu.Lock()
	defer mu.Unlock()

	c = int32(level)
	return Verbose(v >= int32(level))
}

func (v Verbose) Info(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	pairs = append(pairs, "v", c)
	logger.Log(pairs...)
}

func (v Verbose) Infoln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	pairs = append(pairs, "v", c)
	logger.Log(pairs...)
}

func (v Verbose) Infof(format string, args ...interface{}) {
	logger.Log(
		"v", c,
		"msg", fmt.Sprintf(format, args...),
	)
}

func Info(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func InfoDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Infoln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Infof(format string, args ...interface{}) {
	logger.Log("msg", fmt.Sprintf(format, args...))
}

func Warning(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func WarningDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Warningln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Warningf(format string, args ...interface{}) {
	logger.Log("msg", fmt.Sprintf(format, args...))
}

func Error(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func ErrorDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Errorln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Errorf(format string, args ...interface{}) {
	logger.Log("msg", fmt.Sprintf(format, args...))
}

func Fatal(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func FatalDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Fatalln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Log("msg", fmt.Sprintf(format, args...))
}

func Exit(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func ExitDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Exitln(args ...interface{}) {
	var pairs []interface{}
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Exitf(format string, args ...interface{}) {
	logger.Log("msg", fmt.Sprintf(format, args...))
}
