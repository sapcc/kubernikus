package nodeobservatory

import (
	"context"
	"time"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
)

const (
	NodeResyncPeriod = 5 * time.Minute
)

type NodeInformer struct {
	cache.SharedIndexInformer

	kluster *v1.Kluster
	stopCh  chan struct{}
}

func newNodeInformerForKluster(clientFactory kubernetes.SharedClientFactory, kluster *v1.Kluster) (*NodeInformer, error) {
	client, err := clientFactory.ClientFor(kluster)
	if err != nil {
		return nil, err
	}

	nodeInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().Nodes().List(context.TODO(), meta_v1.ListOptions{})
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Nodes().Watch(context.TODO(), meta_v1.ListOptions{})
			},
		},
		&api_v1.Node{},
		NodeResyncPeriod,
		cache.Indexers{},
	)
	return &NodeInformer{
		SharedIndexInformer: nodeInformer,
		kluster:             kluster,
	}, nil
}

func (ni *NodeInformer) run() {
	ni.stopCh = make(chan struct{})
	ni.Run(ni.stopCh)
}

func (ni *NodeInformer) close() {
	close(ni.stopCh)
}
