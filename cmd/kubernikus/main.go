package main

import (
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus"
)

func main() {
	defer glog.Flush()

	baseName := filepath.Base(os.Args[0])

	err := kubernikus.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
