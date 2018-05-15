package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	TestWaitForPodsRunningTimeout      = 5 * time.Minute
	TestWaitForServiceEndpointsTimeout = 5 * time.Minute

	TestPodTimeout             = 1 * time.Minute
	TestServicesTimeout        = 1 * time.Minute
	TestServicesWithDNSTimeout = 1 * time.Minute

	PollInterval = 6 * time.Second // DNS Timeout is 5s

	ServeHostnameImage = "gcr.io/kubernetes-e2e-test-images/serve-hostname-amd64:1.1"
	ServeHostnamePort  = 9376
)

type NetworkTests struct {
	Kubernetes *framework.Kubernetes
	Nodes      *v1.NodeList
	Namespace  string
}

func (n *NetworkTests) CreateNamespace(t *testing.T) {
	_, err := n.Kubernetes.ClientSet.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: n.Namespace}})
	require.NoError(t, err, "There should be no error while creating a namespace")
}

func (n *NetworkTests) WaitForNamespace(t *testing.T) {
	err := n.Kubernetes.WaitForDefaultServiceAccountInNamespace(n.Namespace)
	require.NoError(t, err, "There should be no error while waiting for the namespace")
}

func (n *NetworkTests) DeleteNamespace(t *testing.T) {
	err := n.Kubernetes.ClientSet.CoreV1().Namespaces().Delete(n.Namespace, nil)
	require.NoError(t, err, "There should be no error while deleting a namespace")
}

func (n *NetworkTests) CreatePods(t *testing.T) {
	for _, node := range n.Nodes.Items {
		node := node

		t.Run(node.Name, func(t *testing.T) {
			t.Parallel()

			_, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).Create(&v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-", node.Name),
					Namespace:    n.Namespace,
					Labels: map[string]string{
						"app":  "serve-hostname",
						"node": node.Name,
					},
				},
				Spec: v1.PodSpec{
					NodeName: node.Name,
					Containers: []v1.Container{
						{
							Image: ServeHostnameImage,
							Name:  "serve-hostname",
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: ServeHostnamePort,
								},
							},
						},
					},
				},
			})
			assert.NoError(t, err, "There should be no error while creating a pod")
		})
	}
}

func (n *NetworkTests) WaitForPodsRunning(t *testing.T) {
	label := labels.SelectorFromSet(labels.Set(map[string]string{"app": "serve-hostname"}))
	_, err := n.Kubernetes.WaitForPodsWithLabelRunningReady(n.Namespace, label, len(n.Nodes.Items), TestWaitForPodsRunningTimeout)
	require.NoError(t, err, "Pods must become ready")
}

func (n *NetworkTests) CreateServices(t *testing.T) {
	for _, node := range n.Nodes.Items {
		node := node

		t.Run(node.Name, func(t *testing.T) {
			t.Parallel()

			service := &v1.Service{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      node.Name,
					Namespace: n.Namespace,
					Labels: map[string]string{
						"service": "e2e",
					},
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Port:       ServeHostnamePort,
							TargetPort: intstr.FromInt(ServeHostnamePort),
						},
					},
					Type: v1.ServiceType(v1.ServiceTypeClusterIP),
					Selector: map[string]string{
						"app":  "serve-hostname",
						"node": node.Name,
					},
				},
			}

			_, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).Create(service)
			require.NoError(t, err, "There should be no error while creating a service")
		})
	}
}

func (n *NetworkTests) WaitForServiceEndpoints(t *testing.T) {
	label := labels.SelectorFromSet(labels.Set(map[string]string{"service": "e2e"}))
	_, err := n.Kubernetes.WaitForServiceEndpointsWithLabelNum(n.Namespace, label, 1, TestWaitForServiceEndpointsTimeout)
	require.NoError(t, err, "Services must have 1 endpoint")

}

func (n *NetworkTests) TestPods(t *testing.T) {
	t.Parallel()

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range pods.Items {
		target := target
		t.Run(target.Status.PodIP, func(t *testing.T) {
			t.Parallel()

			for _, source := range pods.Items {
				source := source

				t.Run(source.Status.PodIP, func(t *testing.T) {
					t.Parallel()

					var stdout string
					cmd := strings.Split(fmt.Sprintf("wget -O - http://%v:%v", target.Status.PodIP, ServeHostnamePort), " ")
					err = wait.PollImmediate(PollInterval, TestPodTimeout,
						func() (bool, error) {
							stdout, _, err = n.Kubernetes.ExecCommandInContainerWithFullOutput(n.Namespace, source.Name, source.Spec.Containers[0].Name, cmd...)
							if err != nil {
								return false, nil
							}
							assert.Regexp(t, target.Name, stdout, "should respond with its hostname")
							return true, nil
						})

					assert.NoError(t, err, "Pods should be able to communicate: %s", err)
				})
			}
		})
	}
}

func (n *NetworkTests) TestServices(t *testing.T) {
	t.Parallel()

	services, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).List(meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing services")
	assert.Equal(t, len(n.Nodes.Items), len(services.Items), "There should one service for each node")

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range services.Items {
		target := target
		t.Run(target.Spec.ClusterIP, func(t *testing.T) {
			t.Parallel()

			for _, source := range pods.Items {
				source := source
				t.Run(source.Status.PodIP, func(t *testing.T) {
					t.Parallel()
					var stdout string
					cmd := strings.Split(fmt.Sprintf("wget -O - http://%v:%v", target.Spec.ClusterIP, ServeHostnamePort), " ")
					err = wait.PollImmediate(PollInterval, TestServicesTimeout,
						func() (bool, error) {
							stdout, _, err = n.Kubernetes.ExecCommandInContainerWithFullOutput(n.Namespace, source.Name, source.Spec.Containers[0].Name, cmd...)
							if err != nil {
								return false, nil
							}
							assert.Regexp(t, target.Name, stdout, "should respond with its hostname")
							return true, nil
						})

					assert.NoError(t, err, "Pods should be able to communicate: %s", err)
				})
			}
		})
	}

}

func (n *NetworkTests) TestServicesWithDNS(t *testing.T) {
	t.Parallel()

	services, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).List(meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing services")
	assert.Equal(t, len(n.Nodes.Items), len(services.Items), "There should one service for each node")

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range services.Items {
		target := target
		service := fmt.Sprintf("%s.%s.svc", target.GetName(), target.GetNamespace())

		t.Run(service, func(t *testing.T) {
			t.Parallel()

			for _, source := range pods.Items {
				source := source

				t.Run(source.Status.PodIP, func(t *testing.T) {
					t.Parallel()

					var stdout string
					cmd := strings.Split(fmt.Sprintf("wget -O - http://%v:%v", service, ServeHostnamePort), " ")
					err = wait.PollImmediate(PollInterval, TestServicesWithDNSTimeout,
						func() (bool, error) {
							stdout, _, err = n.Kubernetes.ExecCommandInContainerWithFullOutput(n.Namespace, source.Name, source.Spec.Containers[0].Name, cmd...)
							if err != nil {
								return false, nil
							}

							assert.Regexp(t, target.Name, stdout, "should respond with its hostname")
							return true, nil
						})
					assert.NoError(t, err, "Pods should be able to communicate: %s", err)
				})
			}
		})
	}
}
