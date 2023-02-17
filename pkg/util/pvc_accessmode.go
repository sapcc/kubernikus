package util

import (
	"context"
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

// PVAccessMode is a helper that tries to determine which access mode
// to use for pvc.
// It default to ReadWriteOnce and only returns ReadWriteMany when
// there are no storage classes and at least one ReadWriteMany PV
func PVAccessMode(client kubernetes.Interface, kluster *v1.Kluster) (string, error) {

	if kluster != nil {
		if pvcList, err := client.CoreV1().PersistentVolumeClaims(kluster.Namespace).List(context.TODO(), meta_v1.ListOptions{LabelSelector: fmt.Sprintf("release=%s", kluster.Name)}); err == nil && len(pvcList.Items) > 0 && len(pvcList.Items[0].Spec.AccessModes) > 0 {
			return string(pvcList.Items[0].Spec.AccessModes[0]), nil
		}
	}
	sClasses, err := client.StorageV1().StorageClasses().List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(sClasses.Items) > 0 {
		return string(api_v1.ReadWriteOnce), nil
	}

	pvs, err := client.CoreV1().PersistentVolumes().List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, pv := range pvs.Items {
		for _, mode := range pv.Spec.AccessModes {
			if mode == api_v1.ReadWriteMany {
				return string(api_v1.ReadWriteMany), nil
			}
		}
	}

	return string(api_v1.ReadWriteOnce), nil

}
