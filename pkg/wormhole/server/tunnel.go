package server

import (
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/koding/tunnel"
)

type Tunnel struct {
	Server *tunnel.Server
}

type TunnelOptions struct {
}

func NewTunnel(options *TunnelOptions) *Tunnel {
	server, err := tunnel.NewServer(&tunnel.ServerConfig{})
	if err != nil {
		glog.Fatalf("Failed to create tunnel server: %s", err)
	}

	return &Tunnel{Server: server}
}

func (t *Tunnel) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	glog.Infof("Starting Tunnel Server")
	http.ListenAndServe(":80", t.Server)

	<-stopCh
}
