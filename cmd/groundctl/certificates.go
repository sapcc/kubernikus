package main

import (
	"fmt"

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
		ground.WriteCertificateAuthorities(args[0])
	},
}
