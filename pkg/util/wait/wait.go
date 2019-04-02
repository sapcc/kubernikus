package wait

import (
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

var DoesNotExist = errors.New("The object was not found in the cache")

func WaitForKluster(kluster *v1.Kluster, c cache.Indexer, condition func(cache *v1.Kluster) (bool, error)) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(kluster)
	if err != nil {
		return err
	}
	//Wait for up to 5 seconds for the local cache to reflect the phase change
	return wait.Poll(50*time.Millisecond, 5*time.Second, func() (bool, error) {
		o, exists, err := c.GetByKey(key)
		if !exists {
			return false, DoesNotExist
		}
		if err != nil {
			return false, err
		}
		return condition(o.(*v1.Kluster))
	})
}

func WaitForKlusterDeletion(kluster *v1.Kluster, c cache.Indexer) error {
	if err := WaitForKluster(kluster, c, func(_ *v1.Kluster) (bool, error) { return false, nil }); err != DoesNotExist {
		return err
	}
	return nil

}
