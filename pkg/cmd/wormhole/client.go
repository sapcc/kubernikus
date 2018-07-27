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
	"github.com/sapcc/kubernikus/pkg/wormhole/client"
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

	o.BindFlags(c.Flags())

	return c
}

type ClientOptions struct {
	client.Options
	HealthCheck bool
	NodeName    string
	LogLevel    int
}

func NewClientOptions() *ClientOptions {
	o := &ClientOptions{}
	return o
}

func (o *ClientOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.BoolVar(&o.HealthCheck, "health-check", o.HealthCheck, "Run the health checker (default: true)")
	flags.StringVar(&o.NodeName, "node-name", o.NodeName, "Override the node name used for reporting health")
	flags.StringVar(&o.Context, "context", "", "Override context")
	flags.StringVar(&o.ClientCA, "ca", o.ClientCA, "CA to use for validating tunnel clients")
	flags.StringVar(&o.Certificate, "cert", o.Certificate, "Certificate for the tunnel server")
	flags.StringVar(&o.PrivateKey, "key", o.PrivateKey, "Key for the tunnel server")
	flags.StringVar(&o.ServiceCIDR, "service-cidr", "", "Cluster service IP range")
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *ClientOptions) Validate(c *cobra.Command, args []string) error {
	if o.ServiceCIDR == "" {
		return errors.New("You must specify service-cidr")
	}
	return nil
}

func (o *ClientOptions) Complete(args []string) error {
	return nil
}

func (o *ClientOptions) Run(c *cobra.Command) error {
	o.Logger = logutil.NewLogger(o.LogLevel)
	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
	wg := &sync.WaitGroup{}                            // Goroutines can add themselves to this to be waited on

	client, err := client.New(&o.Options)
	if err != nil {
		return fmt.Errorf("Failed to initialize client: %s", err)
	}

	go client.Run(stop, wg)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	o.Logger.Log("msg", "Shutting down...")
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped

	return nil
}
