package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

type APITests struct {
	Kubernikus  *framework.Kubernikus
	KlusterName string
}

func (a *APITests) Run(t *testing.T) {
	t.Run("ListCluster", a.ListClusters)
	t.Run("ShowCluster", a.ShowCluster)
	t.Run("GetClusterInfo", a.GetClusterInfo)
	t.Run("GetCredentials", a.GetCredentials)
}

func (a *APITests) ListClusters(t *testing.T) {
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
	cluster, err := a.Kubernikus.Client.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	require.NotNil(t, cluster.Payload, "There should be payload")
	assert.Equal(t, a.KlusterName, cluster.Payload.Name, "The shown kluster should have the same name")
}

func (a *APITests) GetClusterInfo(t *testing.T) {
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
	cred, err := a.Kubernikus.Client.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	assert.NotNil(t, cred.Payload.Kubeconfig, "There should be a kubeconfig")
	assert.Contains(t, cred.Payload.Kubeconfig, "clusters", "There should be clusters in the kubeconfig")
}
