package nodeobservatory

import (
	"sync"

	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type Controller interface {
	Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup)

	GetStoreForKluster(*v1.Kluster) cache.Store
	GetIndexerForKluster(*v1.Kluster) cache.Indexer
	HasSyncedForKluster(*v1.Kluster) bool
	List() map[*v1.Kluster]cache.SharedIndexInformer
	AddEventHandlerFuncs(AddFunc, UpdateFunc, DeleteFunc)
}
