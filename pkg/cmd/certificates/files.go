package certificates

import (
	"errors"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"

	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewFilesCommand() *cobra.Command {
	o := NewFilesOptions()

	c := &cobra.Command{
		Use:   "files NAME",
		Short: "Writes certificates to files",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type FilesOptions struct {
	Name string
}

func NewFilesOptions() *FilesOptions {
	return &FilesOptions{}
}

func (o *FilesOptions) BindFlags(flags *pflag.FlagSet) {
}

func (o *FilesOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("you must specify the cluster's name")
	}

	return nil
}

func (o *FilesOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *FilesOptions) Run(c *cobra.Command) error {
	cluster, err := ground.NewCluster(
		&v1.Kluster{
			Spec: v1.KlusterSpec{
				Name: o.Name,
			},
		}, config.Config{
			Kubernikus: config.KubernikusConfig{
				Domain: "local",
			},
		})
	if err != nil {
		return err
	}

	err = cluster.WriteConfig(ground.NewFilePersister("."))
	if err != nil {
		return err
	}

	return nil
}
