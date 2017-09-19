package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/wormhole"
)

func main() {
	defer glog.Flush()
	if f := flag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true") // log to stderr by default
	}

	baseName := filepath.Base(os.Args[0])

	err := wormhole.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
