package main

import (
	"fmt"
	"os"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
)

func init() {
	helmCmd.AddCommand(valuesCmd)
	RootCmd.AddCommand(helmCmd)
}

var helmCmd = &cobra.Command{
	Use: "helm",
}

var valuesCmd = &cobra.Command{
	Use: "values [NAME]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("You need to give a satellite name")
			return
		}

		var result = ""
		cluster, err := ground.NewCluster(args[0])

		if err == nil {
			err = cluster.WriteConfig(ground.NewHelmValuePersister(&result))
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		fmt.Println(result)
	},
}
