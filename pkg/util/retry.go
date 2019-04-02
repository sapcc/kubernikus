package util

import (
	"errors"

	"k8s.io/client-go/util/retry"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	client "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
	listers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
)

type updateKlusterFunc func(kluster *v1.Kluster) error

var KlusterNotUpdated = errors.New("Kluster not updated")

// UpdateKlusterWithRetries updates a kluster with given applyUpdate function.
func UpdateKlusterWithRetries(klusterClient client.KlusterInterface, klusterLister listers_kubernikus.KlusterNamespaceLister, name string, applyUpdate updateKlusterFunc) (*v1.Kluster, error) {
	var kluster *v1.Kluster

	retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var err error
		kluster, err = klusterLister.Get(name)
		if err != nil {
			return err
		}
		kluster = kluster.DeepCopy()
		// Apply the update, then attempt to push it to the apiserver.
		if applyErr := applyUpdate(kluster); applyErr != nil {
			if err == KlusterNotUpdated {
				return nil
			}
			return applyErr
		}
		kluster, err = klusterClient.Update(kluster)
		return err
	})

	return kluster, retryErr
}
