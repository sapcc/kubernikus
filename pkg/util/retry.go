package util

import (
	"context"
	"errors"

	"k8s.io/client-go/util/retry"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	client "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
	listers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			if applyErr == KlusterNotUpdated {
				return nil
			}
			return applyErr
		}
		kluster, err = klusterClient.Update(context.TODO(), kluster, metav1.UpdateOptions{})
		return err
	})

	return kluster, retryErr
}
