package main

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/databus23/guttle"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ClientCommand() *cobra.Command {

	o := ClientOptions{}

	c := &cobra.Command{
		Use:   "client",
		Short: "Starts the guttle client",
		RunE: func(c *cobra.Command, args []string) (err error) {
			if err = o.Validate(); err != nil {
				return
			}

			if o.ClientCert != "" {
				tlsConfig, err2 := newTLSConfig(o.ClientCert, o.ClientKey)
				if err2 != nil {
					return err2
				}
				if o.CA != "" {
					if tlsConfig.RootCAs, err = loadCAFile(o.CA); err != nil {
						return err
					}
				}
				o.ClientOptions.Dial = func(network, addr string) (net.Conn, error) {
					return tls.Dial(network, addr, tlsConfig)
				}
			}

			client := guttle.NewClient(&o.ClientOptions)
			go func() {
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel
				<-sigs
				log.Print("Signal received, closing.")
				client.Stop()
				<-sigs //second signal terminates right away
				os.Exit(1)

			}()
			return client.Start()
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type ClientOptions struct {
	guttle.ClientOptions
	ClientCert string
	ClientKey  string
	CA         string
}

func (o *ClientOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.ServerAddr, "server", "", "Tunnel server address (ip:port)")
	flags.StringVar(&o.ListenAddr, "listen-addr", "", "Listen for incoming connections to send through the tunnel")
	flags.StringVar(&o.ClientCert, "cert", "", "TLS client cert for tunnel connection")
	flags.StringVar(&o.ClientKey, "key", "", "TLS client key for tunnel connection")
	flags.StringVar(&o.CA, "ca", "", "CA for validating tunnel server (default: system chain)")
}

func (o *ClientOptions) Validate() error {
	if o.ServerAddr == "" {
		return errors.New("--server is required")
	}
	return nil
}
