package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	TestKlusterDeletedTimeout    = 5 * time.Minute
	TestKlusterNodesReadyTimeout = 10 * time.Minute

	SmokeTestNodeCount = 2
)

type APITests struct {
	Kubernikus  *framework.Kubernikus
	KlusterName string
}

func (a *APITests) CreateCluster(t *testing.T) {
	kluster := &models.Kluster{
		Name: a.KlusterName,
		Spec: models.KlusterSpec{
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCXIxVEUgtUVkvk2VM1hmIb8MxvxsmvYoiq9OBy3J8akTGNybqKsA2uhcwxSJX5Cn3si8kfMfka9EWiJT+e1ybvtsGILO5XRZPxyhYzexwb3TcALwc3LuzpF3Z/Dg2jYTRELTGhYmyca3mxzTlCjNXvYayLNedjJ8fIBzoCuSXNqDRToHru7h0Glz+wtuE74mNkOiXSvhtuJtJs7VCNVjobFQNfC1aeDsri2bPRHJJZJ0QF4LLYSayMEz3lVwIDyAviQR2Aa97WfuXiofiAemfGqiH47Kq6b8X7j3bOYGBvJKMUV7XeWhGsskAmTsvvnFxkc5PAD3Ct+liULjiQWlzDrmpTE8aMqLK4l0YQw7/8iRVz6gli42iEc2ZG56ob1ErpTLAKFWyCNOebZuGoygdEQaGTIIunAncXg5Rz07TdPl0Tf5ZZLpiAgR5ck0H1SETnjDTZ/S83CiVZWJgmCpu8YOKWyYRD4orWwdnA77L4+ixeojLIhEoNL8KlBgsP9Twx+fFMWLfxMmiuX+yksM6Hu+Lsm+Ao7Q284VPp36EB1rxP1JM7HCiEOEm50Jb6hNKjgN4aoLhG5yg+GnDhwCZqUwcRJo1bWtm3QvRA+rzrGZkId4EY3cyOK5QnYV5+24x93Ex0UspHMn7HGsHUESsVeV0fLqlfXyd2RbHTmDMP6w== Kubernikus Master Key",
			NodePools: []models.NodePool{
				{
					Name:   "small",
					Flavor: "m1.small",
					Image:  "coreos-stable-amd64",
					Size:   SmokeTestNodeCount,
				},
			},
		},
	}

	cluster, err := a.Kubernikus.Client.Operations.CreateCluster(
		operations.NewCreateClusterParams().WithBody(kluster),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	require.NotNil(t, cluster.Payload, "There should be payload")
	assert.Equalf(t, a.KlusterName, cluster.Payload.Name, "There should be a kluster with name: %v", a.KlusterName)
	assert.Equal(t, cluster.Payload.Status.Phase, models.KlusterPhasePending, "Kluster should be in phase Pending")
}

func (a *APITests) ListClusters(t *testing.T) {
	t.Parallel()

	clusterList, err := a.Kubernikus.Client.Operations.ListClusters(nil, a.Kubernikus.AuthInfo)

	found := false
	for _, kluster := range clusterList.Payload {
		if kluster.Name == a.KlusterName {
			found = true
			break
		}
	}

	require.NoError(t, err, "There should be no error")
	assert.NotEmpty(t, clusterList.Payload, "There should be at least one kluster")
	assert.Truef(t, found, "There should be a kluster with name: %v", a.KlusterName)
}

func (a *APITests) ShowCluster(t *testing.T) {
	t.Parallel()

	cluster, err := a.Kubernikus.Client.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	require.NotNil(t, cluster.Payload, "There should be payload")
	assert.Equal(t, a.KlusterName, cluster.Payload.Name, "The shown kluster should have the same name")
}

func (a *APITests) GetClusterInfo(t *testing.T) {
	t.Parallel()

	clusterInfo, err := a.Kubernikus.Client.Operations.GetClusterInfo(
		operations.NewGetClusterInfoParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	assert.NotNil(t, clusterInfo.Payload.SetupCommand, "There should be a setup command")
	for _, v := range clusterInfo.Payload.Binaries {
		assert.NotNil(t, v, "There should be no empty binaries")
	}
}

func (a *APITests) GetCredentials(t *testing.T) {
	t.Parallel()

	cred, err := a.Kubernikus.Client.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	assert.NotNil(t, cred.Payload.Kubeconfig, "There should be a kubeconfig")
	assert.Contains(t, cred.Payload.Kubeconfig, "clusters", "There should be clusters in the kubeconfig")
}

func (a *APITests) TerminateCluster(t *testing.T) {
	_, err := a.Kubernikus.Client.Operations.TerminateCluster(
		operations.NewTerminateClusterParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)
	assert.NoError(t, err, "There should be no error")
}

func (a *APITests) WaitForKlusterToBeDeleted(t *testing.T) {
	err := a.Kubernikus.WaitForKlusterToBeDeleted(a.KlusterName, TestKlusterDeletedTimeout)
	require.NoError(t, err, "There should be no error while waiting %v for the kluster to be deleted", TestKlusterDeletedTimeout)
}

func (a *APITests) WaitForNodesReady(t *testing.T) {
	err := a.Kubernikus.WaitForKlusterToHaveEnoughSchedulableNodes(a.KlusterName, TestKlusterNodesReadyTimeout)
	require.NoError(t, err, "The should be no error while waiting %v for the nodes to become ready", TestKlusterNodesReadyTimeout)
}
