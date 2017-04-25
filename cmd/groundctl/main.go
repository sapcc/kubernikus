package main

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{Use: "groundctl"}

func main() {
	RootCmd.Execute()
}
