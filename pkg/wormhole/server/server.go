package server

import (
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/version"
)

const (
	DEFAULT_RECONCILIATION = 5 * time.Minute
)

type Server struct {
	factory    informers.SharedInformerFactory
	client     kubernetes.Interface
	controller *Controller
	tunnel     *Tunnel

	logger log.Logger
}

func New(options *Options) (*Server, error) {
	s := &Server{logger: log.With(options.Logger, "wormhole", "server")}

	client, err := kube.NewClient(options.KubeConfig, options.Context, options.Logger)
	if err != nil {
		return nil, err
	}

	s.client = client
	s.factory = informers.NewSharedInformerFactory(s.client, DEFAULT_RECONCILIATION)
	s.tunnel, err = NewTunnel(options)
	if err != nil {
		return nil, err
	}
	s.controller = NewController(s.factory.Core().V1().Nodes(), options.ServiceCIDR, s.tunnel.Server, options.Logger)

	return s, nil
}

func (s *Server) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	s.logger.Log(
		"msg", "Starting wormhole server",
		"version", version.GitCommit,
	)

	kube.WaitForServer(s.client, stopCh, s.logger)

	s.factory.Start(stopCh)
	s.factory.WaitForCacheSync(stopCh)

	s.logger.Log(
		"msg", "Cache primed. Ready for Action!",
	)

	go s.controller.Run(1, stopCh, wg)
	go s.tunnel.Run(stopCh, wg)
}
