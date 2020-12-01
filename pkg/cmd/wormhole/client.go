package wormhole

import (
	"github.com/go-kit/kit/log"
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
	KubeConfig  string
	Server      string
	Context     string
	ListenAddr  string
	NodeName    string
	HealthCheck bool
	LogLevel    int
}

func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		ListenAddr:  "198.18.128.1:6443",
		HealthCheck: true,
	}
}

func (o *ClientOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Tunnel Server. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.Server, "server", o.Server, "Tunnel Server endpoint (host:port)")
	flags.StringVar(&o.ListenAddr, "listen", o.ListenAddr, "Listen address for accepting tunnel requests")
	flags.StringVar(&o.Context, "context", o.Context, "Kubeconfig context to use. (default: current-context)")
	flags.StringVar(&o.NodeName, "node-name", o.NodeName, "Override the node name used for reporting health")
	flags.BoolVar(&o.HealthCheck, "health-check", o.HealthCheck, "Run the health checker (default: true)")
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *ClientOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *ClientOptions) Complete(args []string) error {
	return nil
}

func (o *ClientOptions) Run(c *cobra.Command) error {
	logger := logutil.NewLogger(o.LogLevel)
	logger = log.With(logger, "wormhole", "client")

	guttleClient, err := client.New(o.KubeConfig, o.Context, o.Server, o.ListenAddr, logger)
	if err != nil {
		return err
	}

	group := cmd.Runner()
	group.Add(
		func() error {
			return guttleClient.Start()
		},
		func(err error) {
			guttleClient.Stop()
		})

	if o.HealthCheck {
		healthChecker, err := client.NewHealthChecker(o.KubeConfig, o.Context, o.NodeName, logger)
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

	fsnotify, err := client.NewFSNotify(o.KubeConfig)
	closeFSNotify := make(chan struct{}, 0)
	if err != nil {
		return err
	}
	group.Add(
		func() error {
			return fsnotify.Start(closeFSNotify)
		},
		func(err error) {
			close(closeFSNotify)
		})

	return group.Run()
}
