package certificates

import (
	"errors"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/spf13/cobra"
)

func NewPlainCommand() *cobra.Command {
	o := NewPlainOptions()

	c := &cobra.Command{
		Use:   "plain NAME",
		Short: "Prints plain certificates to stdout",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	return c
}

type PlainOptions struct {
	Name string
}

func NewPlainOptions() *PlainOptions {
	return &PlainOptions{}
}

func (o *PlainOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("you must specify the cluster's name")
	}

	return nil
}

func (o *PlainOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *PlainOptions) Run(c *cobra.Command) error {
	kluster, err := kubernikus.NewKlusterFactory().KlusterFor(v1.KlusterSpec{Name: o.Name})
	if err != nil {
		return err
	}

	certificates := util.CreateCertificates(kluster, "https://auth.openstack.com", "kubernikus.cloud.sap")

	if err := NewPlainPersister().WriteConfig(certificates); err != nil {
		return err
	}

	return nil
}
