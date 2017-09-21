package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/koding/tunnel"
	"github.com/koding/tunnel/proto"
)

type Tunnel struct {
	Server  *tunnel.Server
	options *TunnelOptions
}

type TunnelOptions struct {
	ClientCA    string
	Certificate string
	PrivateKey  string
}

func NewTunnel(options *TunnelOptions) *Tunnel {
	server, err := tunnel.NewServer(&tunnel.ServerConfig{})
	if err != nil {
		glog.Fatalf("Failed to create tunnel server: %s", err)
	}

	return &Tunnel{Server: server, options: options}
}

func (t *Tunnel) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	caData, err := ioutil.ReadFile(t.options.ClientCA)
	if err != nil {
		glog.Error("Failed to start tunnel server: Can't read CA file %#v for tunnel server: %s", t.options.ClientCA, err)
		return
	}

	clientCA := x509.NewCertPool()
	if !clientCA.AppendCertsFromPEM(caData) {
		glog.Error("Failed to start tunnel server: No certificates found in ca file.")
		return
	}

	server := http.Server{
		Addr:    ":6553",
		Handler: t,
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  clientCA,
		},
	}

	glog.Infof("Starting tunnel server. Listening on %s", server.Addr)
	go func() {
		err := server.ListenAndServeTLS(t.options.Certificate, t.options.PrivateKey)
		if err != http.ErrServerClosed {
			glog.Errorf("Failed to start tunnel server: %s", err)
		} else {
			glog.Info("Tunnel server stopped")
		}
	}()

	<-stopCh
	server.Close()
}

func (t *Tunnel) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logLine(req, *req.URL, time.Now())
	cn := req.TLS.PeerCertificates[0].Subject.CommonName
	//for ClientIdentifier to be the subjects common name
	req.Header.Set(proto.ClientIdentifierHeader, cn)
	t.Server.ServeHTTP(rw, req)

}

func logLine(req *http.Request, url url.URL, ts time.Time) {
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	uri := url.RequestURI()
	fmt.Printf("%s - %s [%s] \"%s %s %s\"\n",
		host,
		req.TLS.PeerCertificates[0].Subject.CommonName,
		ts.Format("02/Jan/2006:15:04:05 -0700"),
		req.Method,
		uri,
		req.Proto,
	)

}
