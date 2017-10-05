package main

import (
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl"
)

func main() {
	defer glog.Flush()

	baseName := filepath.Base(os.Args[0])

	err := kubernikusctl.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
