package main

import (
	"errors"
	"fmt"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(helmCmd)

	helmCmd.Flags().String("name", "", "Name of the satellite cluster")
	viper.BindPFlag("name", certificatesCmd.Flags().Lookup("name"))
}

var helmCmd = &cobra.Command{
	Use: "values --name NAME",
	RunE: func(cmd *cobra.Command, args []string) error {
		var result = ""

		err := validateHelmInputs()
		if err != nil {
			return err
		}

		cluster, err := ground.NewCluster(viper.GetString("name"))
		if err != nil {
			return err
		}

		err = cluster.WriteConfig(ground.NewHelmValuePersister(&result))
		if err != nil {
			return err
		}

		fmt.Println(result)

		return nil
	},
}

func validateHelmInputs() error {
	if viper.GetString("name") == "" {
		return errors.New("You need to provide a name")
	}

	return nil
}
