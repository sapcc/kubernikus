package wormhole

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/wormhole"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewServerCommand() *cobra.Command {
	o := NewServerOptions()

	c := &cobra.Command{
		Use:   "server",
		Short: "Creates a Wormhole Server",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type ServerOptions struct {
	wormhole.ServerOptions
}

func NewServerOptions() *ServerOptions {
	return &ServerOptions{}
}

func (o *ServerOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.ClientCA, "ca", o.ClientCA, "CA to use for validating tunnel clients")
	flags.StringVar(&o.Certificate, "cert", o.Certificate, "Certificate for the tunnel server")
	flags.StringVar(&o.PrivateKey, "key", o.PrivateKey, "Key for the tunnel server")
}

func (o *ServerOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *ServerOptions) Complete(args []string) error {
	return nil
}

func (o *ServerOptions) Run(c *cobra.Command) error {
	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
	wg := &sync.WaitGroup{}                            // Goroutines can add themselves to this to be waited on

	go wormhole.NewServer(&o.ServerOptions).Run(stop, wg)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	glog.Info("Shutting down...")
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped

	return nil
}
