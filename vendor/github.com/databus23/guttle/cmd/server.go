package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/databus23/guttle"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ServerCommand() *cobra.Command {
	o := ServerFlags{}

	c := &cobra.Command{
		Use:   "server",
		Short: "Starts the guttle server",
		RunE: func(c *cobra.Command, args []string) (err error) {
			if o.Certificate != "" {
				tlsConfig, err2 := newTLSConfig(o.Certificate, o.PrivateKey)
				if err2 != nil {
					return err2
				}
				if o.ClientCA != "" {
					if tlsConfig.ClientCAs, err = loadCAFile(o.ClientCA); err != nil {
						return
					}
					tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
				}
				if o.Listener, err = tls.Listen("tcp", o.ListenAddr, tlsConfig); err != nil {
					return
				}
			} else {
				if o.Listener, err = net.Listen("tcp", o.ListenAddr); err != nil {
					return
				}
			}
			if o.ProxyAddr != "" {
				o.ServerOptions.ProxyFunc = guttle.StaticProxy(o.ProxyAddr)
			}
			server := guttle.NewServer(&o.ServerOptions)
			go func() {
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
				<-sigs
				log.Print("Signal received, closing.")
				server.Close()
			}()
			server.Start()
			return nil
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type ServerFlags struct {
	guttle.ServerOptions
	ListenAddr  string
	Certificate string
	PrivateKey  string
	ClientCA    string
	ProxyAddr   string
}

func (o *ServerFlags) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.ListenAddr, "listen", ":9090", "Tunnel server listen address")
	flags.StringVar(&o.Certificate, "tls-cert", "", "Path to a PEM encoded certificate for the tunnel server listener")
	flags.StringVar(&o.PrivateKey, "tls-key", "", "Path to a PEM encoded private key for the tunnel server listner")
	flags.StringVar(&o.ClientCA, "client-ca", "", "Path to a PEM encoded certifcates for validating connecting clients")
	flags.StringVar(&o.HijackAddr, "hijack-addr", "127.0.0.1:9191", "Listen for REDIRECTED connections on this address")
	flags.StringVar(&o.ProxyAddr, "proxy-addr", "", "proxy forwarded connections to this address")
}

func (o *ServerFlags) Validate() error {
	return nil
}
