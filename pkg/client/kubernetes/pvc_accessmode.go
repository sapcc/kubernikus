package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
)

//PVAccessMode is a helper that tries to determine which access mode
//to use for pvc.
//It default to ReadWriteOnce and only returns ReadWriteMany when
//there are no storage classes and at least one ReadWriteMany PV
func PVAccessMode(client kubernetes.Interface) (string, error) {

	sClasses, err := client.StorageV1().StorageClasses().List(meta_v1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(sClasses.Items) > 0 {
		return string(api_v1.ReadWriteOnce), nil
	}

	pvs, err := client.CoreV1().PersistentVolumes().List(meta_v1.ListOptions{})
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
