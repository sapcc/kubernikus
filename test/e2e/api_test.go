package main

import (
	"sort"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

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
	t.Run("UpdateCluster", a.UpdateCluster)
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

func (a *APITests) UpdateCluster(t *testing.T) {
	cluster, err := a.Kubernikus.Client.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(a.KlusterName),
		a.Kubernikus.AuthInfo,
	)
	require.NoError(t, err, "There should be no error")

	curVersion, err := semver.NewVersion(cluster.Payload.Spec.Version)
	require.NoError(t, err, "There should be no error")

	info, err := a.Kubernikus.Client.Operations.Info(nil)
	require.NoError(t, err, "There should be no error")

	sortedVersions := make([]*semver.Version, len(info.Payload.AvailableClusterVersions))
	for i, r := range info.Payload.AvailableClusterVersions {
		v, err := semver.NewVersion(r)
		if err != nil {
			require.NoError(t, err, "There should be no error")
		}
		sortedVersions[i] = v
	}
	sort.Sort(semver.Collection(sortedVersions))

	toVersion := ""
	for i, vs := range sortedVersions {
		if vs.Equal(curVersion) {
			if len(sortedVersions) > i+1 {
				toVersion = sortedVersions[i+1].String()
				break
			} else {
				// nothing to upgrade to
				return
			}
		}
	}

	params := operations.NewUpdateClusterParams()
	params.SetName(a.KlusterName)
	params.SetBody(cluster.Payload)
	params.Body.Spec.Version = toVersion

	_, err = a.Kubernikus.Client.Operations.UpdateCluster(
		params,
		a.Kubernikus.AuthInfo,
	)
	require.NoError(t, err, "There should be no error")

	newVersion := ""
	err = wait.PollImmediate(5*time.Second, 5*time.Minute,
		func() (bool, error) {
			cluster, err := a.Kubernikus.Client.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(a.KlusterName),
				a.Kubernikus.AuthInfo,
			)
			if err != nil {
				return true, err
			}
			newVersion = cluster.Payload.Status.ApiserverVersion
			return toVersion == newVersion, nil
		})

	require.NoError(t, err, "There should be no error")
	assert.Equal(t, toVersion, newVersion, "The kluster should be updated")
}
