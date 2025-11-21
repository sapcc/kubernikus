package wormhole

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/cmd"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
	"github.com/sapcc/kubernikus/pkg/wormhole/server"
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
	server.Options
	LogLevel int
}

func NewServerOptions() *ServerOptions {
	o := &ServerOptions{}
	return o
}

func (o *ServerOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.Context, "context", "", "Override context")
	flags.StringVar(&o.ClientCA, "ca", o.ClientCA, "CA to use for validating tunnel clients")
	flags.StringVar(&o.Certificate, "cert", o.Certificate, "Certificate for the tunnel server")
	flags.StringVar(&o.PrivateKey, "key", o.PrivateKey, "Key for the tunnel server")
	flags.StringVar(&o.ServiceCIDR, "service-cidr", "", "Cluster service IP range")
	flags.IntVar(&o.ApiPort, "api-port", 6443, "Port the API listens to")
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *ServerOptions) Validate(c *cobra.Command, args []string) error {
	if o.ServiceCIDR == "" {
		return errors.New("you must specify service-cidr")
	}
	return nil
}

func (o *ServerOptions) Complete(args []string) error {
	return nil
}

func (o *ServerOptions) Run(c *cobra.Command) error {
	o.Logger = logutil.NewLogger(o.LogLevel)
	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
	wg := &sync.WaitGroup{}                            // Goroutines can add themselves to this to be waited on

	server, err := server.New(&o.Options)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %s", err)
	}

	go server.Run(stop, wg)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	o.Logger.Log("msg", "Shutting down...")
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped

	return nil
}
