package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func RootCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s is like shuttle without python", name),
	}

	c.AddCommand(
		ServerCommand(),
		ClientCommand(),
	)

	return c
}

func main() {
	baseName := filepath.Base(os.Args[0])
	if err := RootCommand(baseName).Execute(); err != nil {
		log.Fatal(err)
	}

}
