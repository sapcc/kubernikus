package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	wormhole "github.com/sapcc/kubernikus/pkg/wormhole/client"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	// Incremental Increasing TImeout
	StateRunningTimeout                = 5 * time.Minute  // Time from cluster ready to nodes being created
	RegisteredTimeout                  = 15 * time.Minute // Time from node created to registered
	StateSchedulableTimeout            = 1 * time.Minute  // Time from registered to schedulable
	StateHealthyTimeout                = 1 * time.Minute
	ConditionRouteBrokenTimeout        = 1 * time.Minute
	ConditionNetworkUnavailableTimeout = 1 * time.Minute
	ConditionReadyTimeout              = 1 * time.Minute
)

type NodeTests struct {
	Kubernetes        *framework.Kubernetes
	Kubernikus        *framework.Kubernikus
	ExpectedNodeCount int
	KlusterName       string
}

func (k *NodeTests) Run(t *testing.T) {
	_ = t.Run("Created", k.StateRunning) &&
		t.Run("Registered", k.Registered) &&
		t.Run("Schedulable", k.StateSchedulable) &&
		t.Run("NetworkUnavailable", k.ConditionNetworkUnavailable) &&
		t.Run("Healthy", k.StateHealthy) &&
		t.Run("Ready", k.ConditionReady) &&
		t.Run("Labeled", k.Labeled) &&
		t.Run("Sufficient", k.Sufficient)
}

func (k *NodeTests) StateRunning(t *testing.T) {
	count, err := k.checkState(t, func(pool models.NodePoolInfo) int64 { return pool.Running }, StateRunningTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) StateSchedulable(t *testing.T) {
	count, err := k.checkState(t, func(pool models.NodePoolInfo) int64 { return pool.Schedulable }, StateSchedulableTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) StateHealthy(t *testing.T) {
	count, err := k.checkState(t, func(pool models.NodePoolInfo) int64 { return pool.Healthy }, StateHealthyTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) ConditionRouteBroken(t *testing.T) {
	count, err := k.checkCondition(t, wormhole.NodeRouteBroken, v1.ConditionFalse, ConditionRouteBrokenTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) ConditionNetworkUnavailable(t *testing.T) {
	count, err := k.checkCondition(t, v1.NodeNetworkUnavailable, v1.ConditionFalse, ConditionNetworkUnavailableTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) ConditionReady(t *testing.T) {
	count, err := k.checkCondition(t, v1.NodeReady, v1.ConditionTrue, ConditionReadyTimeout)
	assert.NoError(t, err)
	assert.Equal(t, k.ExpectedNodeCount, count)
}

func (k *NodeTests) Labeled(t *testing.T) {
	nodeList, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
	require.NoError(t, err, "There must be no error while listing the kluster's nodes")

	for _, node := range nodeList.Items {
		assert.Contains(t, node.Labels, "ccloud.sap.com/nodepool", "node %s is missing the ccloud.sap.com/nodepool label", node.Name)
	}

}

func (k *NodeTests) Registered(t *testing.T) {
	count := 0
	err := wait.PollImmediate(framework.Poll, RegisteredTimeout,
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

func (k *NodeTests) Sufficient(t *testing.T) {
	nodeList, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
	require.NoError(t, err, "There must be no error while listing the kluster's nodes")
	require.Equal(t, len(nodeList.Items), SmokeTestNodeCount, "There must be exactly %d nodes", SmokeTestNodeCount)
}

type poolCount func(models.NodePoolInfo) int64

func (k *NodeTests) checkState(t *testing.T, fn poolCount, timeout time.Duration) (int, error) {
	count := 0
	err := wait.PollImmediate(framework.Poll, StateRunningTimeout,
		func() (done bool, err error) {
			cluster, err := k.Kubernikus.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(k.KlusterName),
				k.Kubernikus.AuthInfo,
			)
			if err != nil {
				return false, err
			}

			count = int(fn(cluster.Payload.Status.NodePools[0]))
			return count >= k.ExpectedNodeCount, nil
		})

	return count, err
}

func (k *NodeTests) checkCondition(t *testing.T, conditionType v1.NodeConditionType, expectedStatus v1.ConditionStatus, timeout time.Duration) (int, error) {
	count := 0
	err := wait.PollImmediate(framework.Poll, timeout,
		func() (bool, error) {
			nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
			if err != nil {
				return false, fmt.Errorf("Failed to list nodes: %v", err)
			}

			count = 0
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == conditionType {
						if condition.Status == expectedStatus {
							count++
						}
						break
					}
				}
			}

			return count >= k.ExpectedNodeCount, nil
		})

	return count, err
}
