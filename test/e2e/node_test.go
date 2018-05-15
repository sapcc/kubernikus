package main

import (
	"fmt"
	"testing"
	"time"

	wormhole "github.com/sapcc/kubernikus/pkg/wormhole/client"
	"github.com/sapcc/kubernikus/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	TestRegisteredTimeout         = 10 * time.Minute
	TestRouteBrokenTimeout        = 2 * time.Minute
	TestNetworkUnavailableTimeout = 2 * time.Minute
	TestReadyTimeout              = 5 * time.Minute
)

type NodeTests struct {
	Kubernetes        *framework.Kubernetes
	ExpectedNodeCount int
}

func (k *NodeTests) Registered(t *testing.T) {
	count := 0
	err := wait.PollImmediate(framework.Poll, TestRegisteredTimeout,
		func() (bool, error) {
			nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to list nodes: %v", err)
			}
			count = len(nodes.Items)

			return count >= k.ExpectedNodeCount, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) RouteBroken(t *testing.T) {
	t.Parallel()

	count := 0
	err := wait.PollImmediate(framework.Poll, TestRouteBrokenTimeout,
		func() (bool, error) {
			nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to list nodes: %v", err)
			}

			count = 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == wormhole.NodeRouteBroken {
						if condition.Status == v1.ConditionFalse {
							count++
						}
						break
					}
				}
			}

			return count >= k.ExpectedNodeCount, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) NetworkUnavailable(t *testing.T) {
	t.Parallel()

	count := 0
	err := wait.PollImmediate(framework.Poll, TestNetworkUnavailableTimeout,
		func() (bool, error) {
			nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to list nodes: %v", err)
			}

			count = 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == v1.NodeNetworkUnavailable {
						if condition.Status == v1.ConditionFalse {
							count++
						}
						break
					}
				}
			}

			return count >= k.ExpectedNodeCount, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) Ready(t *testing.T) {
	t.Parallel()

	count := 0
	err := wait.PollImmediate(framework.Poll, TestKlusterDeletedTimeout,
		func() (bool, error) {
			nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to list nodes: %v", err)
			}

			count = 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == v1.NodeReady {
						if condition.Status == v1.ConditionTrue {
							count++
						}
						break
					}
				}
			}

			return count >= k.ExpectedNodeCount, nil
		})

	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}
