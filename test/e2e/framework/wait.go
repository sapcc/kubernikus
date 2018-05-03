package framework

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
)

func (f *Kubernetes) WaitForDefaultServiceAccountInNamespace(namespace string) error {
	w, err := f.ClientSet.CoreV1().ServiceAccounts(namespace).Watch(metav1.SingleObject(metav1.ObjectMeta{Name: "default"}))
	if err != nil {
		return err
	}
	_, err = watch.Until(ServiceAccountProvisionTimeout, w, ServiceAccountHasSecrets)
	return err
}

func (f *Kubernetes) WaitForPodsWithLabelRunningReady(ns string, label labels.Selector, num int, timeout time.Duration) (pods *v1.PodList, err error) {
	var current int
	err = wait.Poll(Poll, timeout,
		func() (bool, error) {
			pods, err := f.WaitForPodsWithLabel(ns, label)
			if err != nil {
				if IsRetryableAPIError(err) {
					return false, nil
				}
				return false, fmt.Errorf("Failed to list pods: %v", err)
			}
			current = 0
			for _, pod := range pods.Items {
				if flag, err := PodRunningReady(&pod); err == nil && flag == true {
					current++
				}
			}
			if current != num {
				return false, nil
			}
			return true, nil
		})
	return pods, err
}

func (f *Kubernetes) WaitForPodsWithLabel(ns string, label labels.Selector) (pods *v1.PodList, err error) {
	for t := time.Now(); time.Since(t) < PodListTimeout; time.Sleep(Poll) {
		options := metav1.ListOptions{LabelSelector: label.String()}
		pods, err = f.ClientSet.CoreV1().Pods(ns).List(options)
		if err != nil {
			if IsRetryableAPIError(err) {
				continue
			}
			return
		}
		if len(pods.Items) > 0 {
			break
		}
	}
	if pods == nil || len(pods.Items) == 0 {
		err = fmt.Errorf("Timeout while waiting for pods with label %v", label)
	}
	return
}

func (f *Kubernetes) WaitForServiceEndpointsWithLabelNum(namespace string, label labels.Selector, expectNum int, timeout time.Duration) (services *v1.ServiceList, err error) {
	var current int
	err = wait.Poll(Poll, timeout,
		func() (bool, error) {
			options := metav1.ListOptions{LabelSelector: label.String()}
			services, err := f.ClientSet.CoreV1().Services(namespace).List(options)
			if err != nil {
				if IsRetryableAPIError(err) {
					return false, nil
				}
				return false, fmt.Errorf("Failed to list services: %v", err)
			}

			endpoints, err := f.ClientSet.CoreV1().Endpoints(namespace).List(metav1.ListOptions{})
			if err != nil {
				if IsRetryableAPIError(err) {
					return false, nil
				}
				return false, fmt.Errorf("Failed to list endpoints: %v", err)
			}

			current = 0
			for _, service := range services.Items {
				for _, e := range endpoints.Items {
					if e.Name == service.Name && CountEndpointsNum(&e) == expectNum {
						current++
					}
				}
			}
			if current != len(services.Items) {
				return false, nil
			}

			return true, nil
		})
	return services, err
}

func (f *Kubernikus) WaitForKlusterPhase(klusterName string, expectedPhase models.KlusterPhase, timeout time.Duration) (finalPhase models.KlusterPhase, err error) {
	err = wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			cluster, err := f.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				f.AuthInfo,
			)

			if err != nil {
				return false, fmt.Errorf("Failed to show kluster: %v", err)
			}

			if cluster.Payload == nil {
				return false, fmt.Errorf("API return empty response")
			}

			finalPhase = cluster.Payload.Status.Phase

			return finalPhase == expectedPhase, nil
		})

	return finalPhase, err
}

// WaitForPVCBound waits until the pvc is bound or operation timed out
func (f *Kubernetes) WaitForPVCBound(pvcNs, pvcName string, timeout time.Duration) error {
	w, err := f.ClientSet.CoreV1().PersistentVolumeClaims(pvcNs).Watch(
		metav1.SingleObject(
			metav1.ObjectMeta{
				Name: pvcName,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("failed to watch pvc: %v", err)
	}

	_, err = watch.Until(timeout, w, IsPVCBound)
	if err != nil {
		return err
	}
	return nil
}

// WaitForKlusterToHaveEnoughSchedulableNodes waits until the specified number of nodes equals the number of currently running, healthy, schedulable nodes
func (k *Kubernikus) WaitForKlusterToHaveEnoughSchedulableNodes(klusterName string, timeout time.Duration) error {
	return wait.PollImmediate(Poll, timeout,
		func() (done bool, err error) {
			cluster, err := k.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				k.AuthInfo,
			)
			if err != nil {
				return false, err
			}
			return IsAllNodePoolsOfKlusterReady(cluster.Payload), nil
		},
	)
}

func (k *Kubernikus) WaitForKlusterToBeDeleted(klusterName string, timeout time.Duration) error {
	return wait.PollImmediate(Poll, timeout,
		func() (done bool, err error) {
			_, err = k.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				k.AuthInfo,
			)
			if err != nil {
				switch err.(type) {
				case *operations.ShowClusterDefault:
					result := err.(*operations.ShowClusterDefault)
					return result.Code() == 404, nil
				}
				return false, err
			}
			return false, nil
		},
	)
}
