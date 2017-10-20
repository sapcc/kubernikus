package common

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/glog"
)

func CheckError(err error) {
	if err != nil {
		if err != context.Canceled {
			glog.V(3).Infof("%+v", err)
			fmt.Fprintf(os.Stderr, fmt.Sprintf("An error occurred: %v\n", err))
		}
		os.Exit(1)
	}
}
