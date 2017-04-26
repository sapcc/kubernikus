package main

import (
	"fmt"

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
		ground.NewCluster(args[0]).WriteConfig(ground.NewHelmValuePersister(&result))
		fmt.Println(result)
	},
}
