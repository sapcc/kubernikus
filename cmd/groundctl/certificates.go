package main

import (
	"fmt"
	"os"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
)

var name string

func init() {
	certificatesCmd.AddCommand(generateCmd)
	RootCmd.AddCommand(certificatesCmd)
}

var certificatesCmd = &cobra.Command{
	Use: "certificates",
}

var generateCmd = &cobra.Command{
	Use: "generate [name]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("You need to give a satellite name")
			return
		}
		cluster, err := ground.NewCluster(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		err = cluster.WriteConfig(ground.NewFilePersister("."))
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	},
}
