package client

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

type Options struct {
	Logger      log.Logger
	KubeConfig  string
	Context     string
	ClientCA    string
	Certificate string
	PrivateKey  string
	ServiceCIDR string
}

type Client struct {
	factory    informers.SharedInformerFactory
	client     kubernetes.Interface
	controller *Controller

	logger log.Logger
}

func New(options *Options) (*Client, error) {
	s := &Client{logger: log.With(options.Logger, "wormhole", "server")}

	client, err := kube.NewClient(options.KubeConfig, options.Context, options.Logger)
	if err != nil {
		return nil, err
	}

	s.client = client
	s.factory = informers.NewSharedInformerFactory(s.client, DEFAULT_RECONCILIATION)
	if err != nil {
		return nil, err
	}
	s.controller = NewController(
		s.factory.Core().V1().Nodes(),
		options.ServiceCIDR,
		options.Logger,
		options.ClientCA,
		options.Certificate,
		options.PrivateKey,
	)

	return s, nil
}

func (s *Client) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
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
}
