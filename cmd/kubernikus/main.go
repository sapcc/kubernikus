package main

import (
	"os"
	"path/filepath"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/cmd/kubernikus"
)

func main() {
	baseName := filepath.Base(os.Args[0])
	err := kubernikus.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
