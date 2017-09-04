package kubernikus

import (
	"flag"

	"github.com/sapcc/kubernikus/pkg/cmd/certificates"
	"github.com/sapcc/kubernikus/pkg/cmd/helm"
	"github.com/sapcc/kubernikus/pkg/cmd/operator"
	"github.com/spf13/cobra"
)

func NewCommand(name string) *cobra.Command {
	c := &cobra.Command{
		Use:   name,
		Short: "Kubernetes as a Service",
		Long:  `Kubernikus is a tool for managing Kubernetes clusters on Openstack.`,
	}

	c.AddCommand(
		certificates.NewCommand(),
		helm.NewCommand(),
		operator.NewCommand(),
	)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
