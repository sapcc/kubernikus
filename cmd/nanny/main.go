package main

import (
	goflag "flag"
	"os"
	"path/filepath"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/nanny"
)

func main() {
	if f := goflag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true") // log to stderr by default
	}

	baseName := filepath.Base(os.Args[0])

	err := nanny.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
