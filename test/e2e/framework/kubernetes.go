package framework

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/watch"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
)

const (
	PodListTimeout                 = 1 * time.Minute
	ServiceAccountProvisionTimeout = 2 * time.Minute
	Poll                           = 2 * time.Second
)

type Kubernetes struct {
	ClientSet  *kubernetes.Clientset
	RestConfig *restclient.Config
}

func NewKubernetesFramework(kubernikus *Kubernikus, kluster string) (*Kubernetes, error) {
	credentials, err := kubernikus.Client.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(kluster),
		kubernikus.AuthInfo,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get Kubeconfig: %s", err)
	}

	apiConfig, err := clientcmd.Load([]byte(credentials.Payload.Kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("couldn't parse kubeconfig: %s", err)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't create rest config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't produce clientset: %v", err)
	}

	return &Kubernetes{
		ClientSet:  clientset,
		RestConfig: restConfig,
	}, nil
}

func (f *Kubernetes) WaitForDefaultServiceAccountInNamespace(namespace string) error {
	w, err := f.ClientSet.CoreV1().ServiceAccounts(namespace).Watch(metav1.SingleObject(metav1.ObjectMeta{Name: "default"}))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), ServiceAccountProvisionTimeout)
	defer cancel()

	_, err = watch.UntilWithoutRetry(ctx, w, ServiceAccountHasSecrets)
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = watch.UntilWithoutRetry(ctx, w, IsPVCBound)
	if err != nil {
		return err
	}
	return nil
}
