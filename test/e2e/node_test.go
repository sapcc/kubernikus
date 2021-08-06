package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/extendedserverattributes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
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
	OpenStack         *framework.OpenStack
	ExpectedNodeCount int
	KlusterName       string
}

func (k *NodeTests) Run(t *testing.T) {
	_ = t.Run("Created", k.StateRunning) &&
		t.Run("Tagged", k.Tagged) &&
		t.Run("Registered", k.Registered) &&
		t.Run("LatestStableContainerLinux", k.LatestStableContainerLinux) &&
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

func (k *NodeTests) Tagged(t *testing.T) {
	instances, err := k.listInstances()
	if err != nil {
		require.NoErrorf(t, err, "listing openstack instances failed")
	}
	assert.Len(t, instances, k.ExpectedNodeCount, "Didn't find expected number of cloud instances")
	for _, instance := range instances {
		assert.Subset(t, *instance.Tags, []string{"kubernikus", "kubernikus:kluster=" + k.KlusterName, "kubernikus:nodepool=small"})

		expect := map[string]string{
			"provisioner":         "kubernikus",
			"kubernikus:kluster":  k.KlusterName,
			"kubernikus:nodepool": "small",
		}
		for k, v := range expect {
			assert.Equalf(t, v, instance.Metadata[k], "metadata key %s incorrect", k)
		}

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

func (k NodeTests) LatestStableContainerLinux(t *testing.T) {
	nodes, err := k.Kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if !assert.NoError(t, err) {
		return
	}

	type FlatcarReleases struct {
		Current struct {
			Channel       string   `json:"channel"`
			Architectures []string `json:"architectures"`
			ReleaseDate   string   `json:"release_date"`
			MajorSoftware struct {
				Docker   []string `json:"docker"`
				Ignition []string `json:"ignition"`
				Kernel   []string `json:"kernel"`
				Systemd  []string `json:"systemd"`
			} `json:"major_software"`
			ReleaseNotes string `json:"release_notes"`
		} `json:"current"`
	}

	resp, err := http.Get("https://kinvolk.io/flatcar-container-linux/releases-json/releases-stable.json")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, 200, resp.StatusCode) {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if !assert.NoError(t, err) {
		return
	}
	var current FlatcarReleases
	err = json.Unmarshal(body, &current)
	if !assert.NoError(t, err) {
		return
	}

	if current.Current.ReleaseDate != "" {
		date, err := time.Parse("2006-01-02", current.Current.ReleaseDate[0:10])
		if !assert.NoError(t, err) {
			return
		}
		if !assert.NotEmpty(t, date, "Could not get release date") {
			return
		}
		// check if release is at least 3 days old, otherwise image might not be up-to-date
		if time.Since(date).Hours() < 72 {
			return
		}
	}

	resp, err = http.Get("https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, 200, resp.StatusCode) {
		return
	}
	version := ""
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		keyval := strings.Split(scanner.Text(), "=")

		if len(keyval) == 2 && keyval[0] == "FLATCAR_VERSION" {
			version = keyval[1]
			if !assert.NotEmpty(t, version, "Failed to detect latest stable Container Linux version") {
				return
			}
		}
	}

	if !assert.NotEmpty(t, version, "Failed to detect latest stable Container Linux version") {
		return
	}

	for _, node := range nodes.Items {
		assert.Contains(t, node.Status.NodeInfo.OSImage, version, "Node %s is not on latest version", node.Name)
	}
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

type instance struct {
	servers.Server
	extendedserverattributes.ServerAttributesExt
}

func (k *NodeTests) listInstances() ([]instance, error) {

	serversListOpts := servers.ListOpts{
		Name: "kks-" + k.KlusterName,
	}

	allPages, err := servers.List(k.OpenStack.Compute, serversListOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error while listing all servers: %w", err)
	}

	var s []instance

	err = servers.ExtractServersInto(allPages, &s)
	if err != nil {
		return nil, fmt.Errorf("error extracting server info: %w", err)
	}
	return s, nil

}
