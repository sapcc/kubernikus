package wormhole

import (
	"sync"
	"time"

	"github.com/go-kit/kit/log"
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

	Logger log.Logger
}

type Server struct {
	factory    informers.SharedInformerFactory
	client     kubernetes.Interface
	controller *server.Controller
	tunnel     *server.Tunnel

	Logger log.Logger
}

func NewServer(options *ServerOptions) (*Server, error) {
	s := &Server{Logger: log.With(options.Logger, "wormhole", "server")}

	client, err := kube.NewClient(options.KubeConfig, options.Context, options.Logger)
	if err != nil {
		return nil, err
	}

	s.client = client
	s.factory = informers.NewSharedInformerFactory(s.client, DEFAULT_RECONCILIATION)
	s.tunnel, err = server.NewTunnel(&options.TunnelOptions)
	if err != nil {
		return nil, err
	}
	s.controller = server.NewController(s.factory.Core().V1().Nodes(), options.ServiceCIDR, s.tunnel.Server, options.Logger)

	return s, nil
}

func (s *Server) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	s.Logger.Log(
		"msg", "powering up wormhole generator",
		"version", version.GitCommit,
	)

	kube.WaitForServer(s.client, stopCh, s.Logger)

	s.factory.Start(stopCh)
	s.factory.WaitForCacheSync(stopCh)

	s.Logger.Log(
		"msg", "Cache primed. Ready for Action!",
	)

	go s.controller.Run(1, stopCh, wg)
	go s.tunnel.Run(stopCh, wg)
}
