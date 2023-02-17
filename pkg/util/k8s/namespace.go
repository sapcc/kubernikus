package k8s

import (
	"context"

	api_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureNamespace(client kubernetes.Interface, ns string) error {
	if _, err := client.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{}); err == nil {
		return nil
	}
	newNs := &api_v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	if _, err := client.CoreV1().Namespaces().Create(context.TODO(), newNs, metav1.CreateOptions{}); !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
