package wormhole

import (
	"github.com/go-kit/kit/log"
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
	Kubeconfig             string
	NodeName               string
	HealthCheck            bool
	ContainerInterfaceName string
	LogLevel               int
}

func NewServerOptions() *ServerOptions {
	o := &ServerOptions{}
	return o
}

func (o *ServerOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.ClientCA, "ca", o.ClientCA, "CA to use for validating tunnel clients")
	flags.StringVar(&o.Certificate, "cert", o.Certificate, "Certificate for the tunnel server")
	flags.StringVar(&o.PrivateKey, "key", o.PrivateKey, "Key for the tunnel server")
	flags.StringVar(&o.NodeName, "node-name", o.NodeName, "Override the node name used for reporting health")
	flags.BoolVar(&o.HealthCheck, "health-check", o.HealthCheck, "Run the health checker (default: true)")
	flags.StringVar(&o.ContainerInterfaceName, "container-interface-name", o.ContainerInterfaceName, "Container interface name to use for healthchecks")
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *ServerOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *ServerOptions) Complete(args []string) error {
	return nil
}

func (o *ServerOptions) Run(c *cobra.Command) error {
	logger := logutil.NewLogger(o.LogLevel)
	logger = log.With(logger, "wormhole", "Server")

	guttleServer, err := server.New(&o.Options)
	if err != nil {
		return err
	}

	group := cmd.Runner()
	group.Add(
		func() error {
			return guttleServer.Start()
		},
		func(err error) {
			guttleServer.Close()
		})

	if o.HealthCheck {
		healthChecker, err := server.NewHealthChecker(o.KubeConfig, o.Context, o.NodeName, o.ContainerInterfaceName, logger)
		if err != nil {
			return err
		}
		closeCh := make(chan struct{}, 0)

		group.Add(
			func() error {
				return healthChecker.Start(closeCh)
			},
			func(err error) {
				close(closeCh)
			})
	}

	return group.Run()
}
