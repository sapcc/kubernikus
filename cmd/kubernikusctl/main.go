package main

import (
	goflag "flag"
	"os"
	"path/filepath"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikusctl"
)

func main() {
	if f := goflag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true")
	}

	baseName := filepath.Base(os.Args[0])

	err := kubernikusctl.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
