package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	TestWaitForPodsRunningTimeout      = 5 * time.Minute
	TestWaitForKubeDNSRunningTimeout   = 5 * time.Minute
	TestWaitForServiceEndpointsTimeout = 5 * time.Minute

	TestPodTimeout                 = 1 * time.Minute
	TestServicesTimeout            = 1 * time.Minute
	TestServicesWithDNSTimeout     = 2 * time.Minute
	TestServiceLoadbalancerTimeout = 10 * time.Minute

	PollInterval = 6 * time.Second // DNS Timeout is 5s

	ServeHostnameImage = "keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/sapcc/serve-hostname-amd64:1.2-alpine"
	ServeHostnamePort  = 9376
)

type NetworkTests struct {
	Kubernetes *framework.Kubernetes
	Namespace  string
	Nodes      *v1.NodeList
}

func (n *NetworkTests) Run(t *testing.T) {
	runParallel(t)

	n.Namespace = generator.SimpleNameGenerator.GenerateName("e2e-network-")

	var err error
	n.Nodes, err = n.Kubernetes.ClientSet.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{})
	require.NoError(t, err, "There must be no error while listing the kluster's nodes")
	require.NotEmpty(t, n.Nodes.Items, "No nodes returned by list")

	defer t.Run("Cleanup", n.DeleteNamespace)
	t.Run("CreateNamespace", n.CreateNamespace)
	t.Run("WaitNamespace", n.WaitForNamespace)
	n.CreatePods(t)
	n.CreateServices(t)
	t.Run("CreateLoadbalancer", n.CreateLoadbalancer)
	t.Run("Wait", func(t *testing.T) {
		t.Run("Pods", n.WaitForPodsRunning)
		t.Run("ServiceEndpoints", n.WaitForServiceEndpoints)
		t.Run("KubeDNS", n.WaitForKubeDNSRunning)
	})
	t.Run("Connectivity", func(t *testing.T) {
		t.Run("Pods", n.TestPods)
		t.Run("Services", n.TestServices)
		t.Run("ServicesWithDNS", n.TestServicesWithDNS)
		t.Run("Loadbalancer", n.TestLoadbalancer)
	})
}

func (n *NetworkTests) CreateNamespace(t *testing.T) {
	_, err := n.Kubernetes.ClientSet.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: n.Namespace}}, meta_v1.CreateOptions{})
	require.NoError(t, err, "There should be no error while creating a namespace")
}

func (n *NetworkTests) WaitForNamespace(t *testing.T) {
	err := n.Kubernetes.WaitForDefaultServiceAccountInNamespace(n.Namespace)
	require.NoError(t, err, "There should be no error while waiting for the namespace")
}

func (n *NetworkTests) DeleteNamespace(t *testing.T) {
	err := n.Kubernetes.ClientSet.CoreV1().Namespaces().Delete(context.Background(), n.Namespace, meta_v1.DeleteOptions{})
	require.NoError(t, err, "There should be no error while deleting a namespace")
}

func (n *NetworkTests) CreatePods(t *testing.T) {
	for _, node := range n.Nodes.Items {
		node := node

		t.Run(fmt.Sprintf("CreatePodForNode-%v", node.Name), func(t *testing.T) {
			_, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).Create(context.Background(), &v1.Pod{
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
			}, meta_v1.CreateOptions{})
			assert.NoError(t, err, "There should be no error while creating a pod")
		})
	}
}

func (n *NetworkTests) WaitForPodsRunning(t *testing.T) {
	runParallel(t)

	label := labels.SelectorFromSet(labels.Set(map[string]string{"app": "serve-hostname"}))
	_, err := n.Kubernetes.WaitForPodsWithLabelRunningReady(n.Namespace, label, len(n.Nodes.Items), TestWaitForPodsRunningTimeout)
	assert.NoError(t, err, "Pods must become ready")
}

func (n *NetworkTests) WaitForKubeDNSRunning(t *testing.T) {
	runParallel(t)

	label := labels.SelectorFromSet(labels.Set(map[string]string{"k8s-app": "kube-dns"}))
	_, err := n.Kubernetes.WaitForPodsWithLabelRunningReady("kube-system", label, 2, TestWaitForKubeDNSRunningTimeout)
	assert.NoError(t, err, "Kube-DNS must become ready")
}

func (n *NetworkTests) CreateServices(t *testing.T) {
	for _, node := range n.Nodes.Items {
		node := node

		t.Run(fmt.Sprintf("CreateServiceForNode-%v", node.Name), func(t *testing.T) {
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

			_, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).Create(context.Background(), service, meta_v1.CreateOptions{})
			require.NoError(t, err, "There should be no error while creating a service")
		})
	}
}

func (n *NetworkTests) WaitForServiceEndpoints(t *testing.T) {
	runParallel(t)

	label := labels.SelectorFromSet(labels.Set(map[string]string{"service": "e2e"}))
	_, err := n.Kubernetes.WaitForServiceEndpointsWithLabelNum(n.Namespace, label, 1, TestWaitForServiceEndpointsTimeout)
	require.NoError(t, err, "Services must have 1 endpoint")

}

func (n *NetworkTests) TestPods(t *testing.T) {
	runParallel(t)

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(context.Background(), meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range pods.Items {
		target := target

		for _, source := range pods.Items {
			source := source

			t.Run(fmt.Sprintf("%v->%v", source.Status.PodIP, target.Status.PodIP), func(t *testing.T) {
				var stdout string
				cmd := strings.Split(fmt.Sprintf("curl -f --max-time 5 http://%v:%v", target.Status.PodIP, ServeHostnamePort), " ")
				err = wait.PollImmediate(PollInterval, TestPodTimeout, //nolint:staticcheck
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
	}
}

func (n *NetworkTests) TestServices(t *testing.T) {
	runParallel(t)

	services, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).List(context.Background(), meta_v1.ListOptions{LabelSelector: "service=e2e"})
	assert.NoError(t, err, "There should be no error while listing services")
	assert.Equal(t, len(n.Nodes.Items), len(services.Items), "There should one service for each node")

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(context.Background(), meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range services.Items {
		target := target

		for _, source := range pods.Items {
			source := source

			t.Run(fmt.Sprintf("%v->%v", source.Status.PodIP, target.Spec.ClusterIP), func(t *testing.T) {
				var stdout string
				cmd := strings.Split(fmt.Sprintf("curl -f --max-time 5 http://%v:%v", target.Spec.ClusterIP, ServeHostnamePort), " ")
				err = wait.PollImmediate(PollInterval, TestServicesTimeout, //nolint:staticcheck
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
	}

}

func (n *NetworkTests) TestServicesWithDNS(t *testing.T) {
	runParallel(t)

	services, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).List(context.Background(), meta_v1.ListOptions{LabelSelector: "service=e2e"})
	assert.NoError(t, err, "There should be no error while listing services")
	assert.Equal(t, len(n.Nodes.Items), len(services.Items), "There should one service for each node")

	pods, err := n.Kubernetes.ClientSet.CoreV1().Pods(n.Namespace).List(context.Background(), meta_v1.ListOptions{})
	assert.NoError(t, err, "There should be no error while listing the kluster's pods")
	assert.Equal(t, len(n.Nodes.Items), len(pods.Items), "There should one pod for each node")

	for _, target := range services.Items {
		target := target
		service := fmt.Sprintf("%s.%s.svc", target.GetName(), target.GetNamespace())

		for _, source := range pods.Items {
			source := source

			t.Run(fmt.Sprintf("%v->%v", source.Status.PodIP, service), func(t *testing.T) {
				var stdout string
				cmd := []string{"dig", service}
				err = wait.PollImmediate(PollInterval, TestServicesWithDNSTimeout, //nolint:staticcheck
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
	}
}

func (n *NetworkTests) CreateLoadbalancer(t *testing.T) {
	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "e2e-lb",
			Namespace: n.Namespace,
			Labels: map[string]string{
				"service": "e2e-lb",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:       ServeHostnamePort,
					TargetPort: intstr.FromInt(ServeHostnamePort),
				},
			},
			Type: v1.ServiceType(v1.ServiceTypeLoadBalancer),
			Selector: map[string]string{
				"app":  "serve-hostname",
				"node": n.Nodes.Items[0].Name,
			},
			ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyTypeCluster,
		},
	}

	_, err := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).Create(context.Background(), service, meta_v1.CreateOptions{})
	require.NoError(t, err, "There should be no error while creating a loadbalancer")
}

func (n *NetworkTests) TestLoadbalancer(t *testing.T) {
	runParallel(t)

	err := wait.PollImmediate(PollInterval, TestServiceLoadbalancerTimeout, //nolint:staticcheck
		func() (bool, error) {
			lb, _ := n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).Get(context.Background(), "e2e-lb", meta_v1.GetOptions{})

			if len(lb.Status.LoadBalancer.Ingress) > 0 && net.ParseIP(lb.Status.LoadBalancer.Ingress[0].IP) != nil {
				return true, nil
			}

			return false, nil
		})
	assert.NoError(t, err, "Loadbalancers should get an external IP: %s", err)

	err = n.Kubernetes.ClientSet.CoreV1().Services(n.Namespace).Delete(context.Background(), "e2e-lb", meta_v1.DeleteOptions{})
	assert.NoError(t, err, "There should be no error deleting loadbalancer service: %s", err)
}
