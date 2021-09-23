package common

import (
	"context"
	"fmt"
	"os"

	"k8s.io/klog"
)

func CheckError(err error) {
	if err != nil {
		if err != context.Canceled {
			klog.V(3).Infof("%+v", err)
			fmt.Fprintf(os.Stderr, "An error occurred: %v\n", err)
		}
		os.Exit(1)
	}
}
