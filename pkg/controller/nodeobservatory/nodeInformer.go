package nodeobservatory

import (
	"fmt"
	"time"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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
				return client.CoreV1().Nodes().List(meta_v1.ListOptions{})
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Nodes().Watch(meta_v1.ListOptions{})
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
	ni.SharedIndexInformer.Run(ni.stopCh)
}

func (ni *NodeInformer) close() {
	close(ni.stopCh)
}

func (ni *NodeInformer) getNodeByKey(key string) (*api_v1.Node, error) {
	obj, exists, err := ni.SharedIndexInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("node %s in kluster %s/%s not found", key, ni.kluster.GetNamespace(), ni.kluster.GetName())
	}
	return obj.(*api_v1.Node), nil
}
