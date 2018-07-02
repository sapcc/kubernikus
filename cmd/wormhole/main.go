package main

import (
	"os"
	"path/filepath"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/wormhole"
)

func main() {
	baseName := filepath.Base(os.Args[0])

	err := wormhole.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
