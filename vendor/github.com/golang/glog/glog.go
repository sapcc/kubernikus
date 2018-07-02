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
	pairs = append(pairs, "v", c)
	pairs = append(pairs, "glog", "Info")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func (v Verbose) Infoln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "v", c)
	pairs = append(pairs, "glog", "Infoln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func (v Verbose) Infof(format string, args ...interface{}) {
	logger.Log(
		"v", c,
		"glog", "Infof",
		"msg", fmt.Sprintf(format, args...),
	)
}

func Info(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Info")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func InfoDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "InfoDepth")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Infoln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Infoln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Infof(format string, args ...interface{}) {
	logger.Log(
		"glog", "Infof",
		"msg", fmt.Sprintf(format, args...),
	)
}

func Warning(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Warning")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func WarningDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "WarningDepth")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Warningln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Warningln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Warningf(format string, args ...interface{}) {
	logger.Log(
		"glog", "Warningf",
		"msg", fmt.Sprintf(format, args...),
	)
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
	pairs = append(pairs, "glog", "ErrorDepth")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Errorln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Errorln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Errorf(format string, args ...interface{}) {
	logger.Log(
		"glog", "Errorf",
		"msg", fmt.Sprintf(format, args...),
	)
}

func Fatal(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Fatal")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func FatalDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "FatalDepth")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Fatalln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Fatalln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Log(
		"glog", "Fatalf",
		"msg", fmt.Sprintf(format, args...),
	)
}

func Exit(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Exit")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func ExitDepth(depth int, args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "ExitDepth")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Exitln(args ...interface{}) {
	var pairs []interface{}
	pairs = append(pairs, "glog", "Exitln")
	for key, value := range args {
		pairs = append(pairs, key, value)
	}
	logger.Log(pairs...)
}

func Exitf(format string, args ...interface{}) {
	logger.Log(
		"glog", "Exitf",
		"msg", fmt.Sprintf(format, args...),
	)
}
