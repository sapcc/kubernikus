package main

import (
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(helmCmd)

	helmCmd.Flags().String("name", "", "Name of the satellite cluster")
	viper.BindPFlag("name", helmCmd.Flags().Lookup("name"))
}

var helmCmd = &cobra.Command{
	Use: "values --name NAME",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateHelmInputs()
		if err != nil {
			return err
		}

		cluster, err := ground.NewCluster(viper.GetString("name"))
		if err != nil {
			return err
		}

		result, err := yaml.Marshal(cluster)
		if err != nil {
			return err
		}

		fmt.Println(string(result))

		return nil
	},
}

func validateHelmInputs() error {
	if viper.GetString("name") == "" {
		return errors.New("You need to provide a name")
	}

	return nil
}
