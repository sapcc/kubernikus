package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	caCmd.AddCommand(etcdCmd)
	certificatesCmd.AddCommand(caCmd)
	RootCmd.AddCommand(certificatesCmd)
}

var certificatesCmd = &cobra.Command{
	Use: "certificates",
}

var caCmd = &cobra.Command{
	Use: "generate",
}

var etcdCmd = &cobra.Command{
	Use: "etcd",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generated CA certificates for etcd")
	},
}
