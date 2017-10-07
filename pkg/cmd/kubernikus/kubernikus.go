package kubernikus

import (
	"flag"

	"github.com/spf13/cobra"
)

func NewCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: "Kubernetes as a Service",
		Long:  `Kubernikus is a tool for managing Kubernetes clusters on Openstack.`,
	}

	c.AddCommand(
		NewCertificatesCommand(),
		NewHelmCommand(),
		NewOperatorCommand(),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
