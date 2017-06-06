package main

import (
	"errors"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(certificatesCmd)

	certificatesCmd.Flags().String("name", "", "Name of the satellite cluster")
	viper.BindPFlag("name", certificatesCmd.Flags().Lookup("name"))
}

var certificatesCmd = &cobra.Command{
	Use: "certificates --name NAME",

	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateCertificateInputs()
		if err != nil {
			return err
		}

		cluster, err := ground.NewCluster(viper.GetString("name"))
		if err != nil {
			return err
		}

		err = cluster.WriteConfig(ground.NewFilePersister("."))
		if err != nil {
			return err
		}

		return nil
	},
}

func validateCertificateInputs() error {
	if viper.GetString("name") == "" {
		return errors.New("You need to provide a name")
	}

	return nil
}
