package wormhole

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/databus23/guttle"
	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/sapcc/kubernikus/pkg/cmd"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
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
	KubeConfig string
	Server     string
	Context    string
	ListenAddr string
}

func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		ListenAddr: "198.18.128.1:6443",
	}
}

func (o *ClientOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Tunnel Server. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.Server, "server", o.Server, "Tunnel Server endpoint (host:port)")
	flags.StringVar(&o.ListenAddr, "listen", o.ListenAddr, "Listen address for accepting tunnel requests")
	flags.StringVar(&o.Context, "context", o.Context, "Kubeconfig context to use. (default: current-context)")
}

func (o *ClientOptions) Validate(c *cobra.Command, args []string) error {
	return nil
}

func (o *ClientOptions) Complete(args []string) error {
	return nil
}

func (o *ClientOptions) Run(c *cobra.Command) error {
	logger := logutil.NewLogger(c.Flags())
	logger = log.With(logger, "wormhole", "client")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel

	config, err := clientcmd.LoadFromFile(o.KubeConfig)
	if err != nil {
		return fmt.Errorf("Failed to load kubeconfig file %#v: %s", o.KubeConfig, err)
	}
	err = api.FlattenConfig(config)
	if err != nil {
		return err
	}
	contextName := config.CurrentContext
	if contextName == "" {
		contextName = o.Context
	}
	if contextName == "" {
		return fmt.Errorf("No context given")
	}

	context, found := config.Contexts[contextName]
	if !found {
		return fmt.Errorf("Context %s not found", contextName)
	}

	cluster, found := config.Clusters[context.Cluster]
	if !found {
		return fmt.Errorf("Cluster not found %s", context.Cluster)
	}

	authInfo, found := config.AuthInfos[context.AuthInfo]
	if !found {
		return fmt.Errorf("No auth info found for context %s", context.AuthInfo)
	}
	cert := authInfo.ClientCertificateData
	key := authInfo.ClientKeyData

	ca := cluster.CertificateAuthorityData

	var rootCAs *x509.CertPool
	if ca != nil {
		rootCAs = x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(ca) {
			return fmt.Errorf("Failed to load any certs from %s", ca)
		}
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("Failed to load certificate/key: %s", err)
	}

	serverAddr := o.Server
	if serverAddr == "" {
		url, err := url.Parse(cluster.Server)
		if err != nil {
			return err
		}
		c := strings.Split(url.Hostname(), ".")
		//Add "-t" to first component of hostname
		c[0] = fmt.Sprintf("%s-wormhole", c[0])
		serverAddr = fmt.Sprintf("%s:%s", strings.Join(c, "."), "443")
	}

	opts := guttle.ClientOptions{
		ServerAddr: serverAddr,
		ListenAddr: o.ListenAddr,
		Dial: func(network, address string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := tls.DialWithDialer(dialer, network, address, &tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{certificate},
			})
			if err != nil {
				logger.Log(
					"msg", "failed to open connection",
					"address", address,
					"err", err)
			}
			return conn, err
		},
	}

	client := guttle.NewClient(&opts)

	go func() {
		<-sigs
		logger.Log("msg", "Shutting down...")
		client.Stop()
	}()
	return client.Start()

	return nil
}
