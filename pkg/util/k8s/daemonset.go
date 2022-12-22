package k8s

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func UpdateDaemonsetImages(client kubernetes.Interface, name, namespace string, containerAndImage ...string) error {
	if l := len(containerAndImage); l < 2 || l%2 == 1 {
		return errors.New("containerAndImage needs to be even number of arguments")
	}
	retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		ds, err := client.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Failed to find daemonset %s/%s: %w", namespace, name, err)
		}
		updated := false
		for i := 0; i < len(containerAndImage)-1; i += 2 {
			container := containerAndImage[i]
			newimage := containerAndImage[i+1]

			for i, c := range ds.Spec.Template.Spec.Containers {
				if c.Name == container && c.Image != newimage {
					ds.Spec.Template.Spec.Containers[i].Image = newimage
					updated = true
				}
			}
		}
		if updated {
			_, err := client.AppsV1().DaemonSets(namespace).Update(context.TODO(), ds, metav1.UpdateOptions{})
			return err
		}
		return nil
	})
	return retryErr
}
