package wormhole

import (
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/spf13/cobra"
)

func NewClientCommand() *cobra.Command {
	o := NewClientOptions()

	c := &cobra.Command{
		Use:   "client",
		Short: "Creates a Wormhole Client",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	return c
}

type ClientOptions struct {
}

func NewClientOptions() *ClientOptions {
	return &ClientOptions{}
}

func (o *ClientOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *ClientOptions) Complete(args []string) error {
	return nil
}

func (o *ClientOptions) Run(c *cobra.Command) error {
	return nil
}
