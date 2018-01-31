package nodeobservatory

import (
	"sync"

	"github.com/go-kit/kit/log"

	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
)

func NewInformerFactory(informer kubernikus_informers_v1.KlusterInformer, clients *kube.SharedClientFactory, logger log.Logger) *InformerFactory {
	return &InformerFactory{
		informer:      informer,
		clientFactory: clients,
		logger:        logger,
	}
}

type InformerFactory struct {
	lock          sync.Mutex
	observatory   *NodeObservatory
	informer      kubernikus_informers_v1.KlusterInformer
	clientFactory *kube.SharedClientFactory
	logger        log.Logger
}

func (f *InformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.observatory != nil {
		go f.observatory.Run(stopCh)
	}
}

func (f *InformerFactory) NodeInformer() *NodeObservatory {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.observatory != nil {
		return f.observatory
	}
	f.observatory = NewController(f.informer, f.clientFactory, f.logger, 10)
	return f.observatory
}
