package wormhole

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/version"
	"github.com/sapcc/kubernikus/pkg/wormhole/server"
)

const (
	DEFAULT_RECONCILIATION = 5 * time.Minute
)

type ServerOptions struct {
	KubeConfig  string
	Context     string
	ServiceCIDR string
	server.TunnelOptions
}

type Server struct {
	factory    informers.SharedInformerFactory
	client     kubernetes.Interface
	controller *server.Controller
	tunnel     *server.Tunnel
}

func NewServer(options *ServerOptions) (*Server, error) {
	s := &Server{}

	client, err := kube.NewClient(options.KubeConfig, options.Context)
	if err != nil {
		return nil, err
	}

	s.client = client
	s.factory = informers.NewSharedInformerFactory(s.client, DEFAULT_RECONCILIATION)
	s.tunnel, err = server.NewTunnel(&options.TunnelOptions)
	if err != nil {
		return nil, err
	}
	s.controller = server.NewController(s.factory.Core().V1().Nodes(), options.ServiceCIDR, s.tunnel.Server)

	return s, nil
}

func (s *Server) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Wormhole %v\n", version.GitCommit)

	kube.WaitForServer(s.client, stopCh)

	s.factory.Start(stopCh)
	s.factory.WaitForCacheSync(stopCh)

	glog.Info("Cache primed. Ready for Action!")

	go s.controller.Run(1, stopCh, wg)
	go s.tunnel.Run(stopCh, wg)
}
