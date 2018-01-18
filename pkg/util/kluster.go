package util

import (
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	clientset "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
)

func EnsureFinalizerCreated(client clientset.KubernikusV1Interface, kluster *v1.Kluster, finalizer string) (err error) {
	if kluster.NeedsFinalizer(finalizer) {
		copy := kluster.DeepCopy()
		copy.AddFinalizer(finalizer)
		_, err = client.Klusters(copy.Namespace).Update(copy)
	}
	return err
}

func EnsureFinalizerRemoved(client clientset.KubernikusV1Interface, kluster *v1.Kluster, finalizer string) (err error) {
	if kluster.HasFinalizer(finalizer) {
		copy := kluster.DeepCopy()
		copy.RemoveFinalizer(finalizer)
		_, err = client.Klusters(copy.Namespace).Update(copy)
	}
	return err
}
