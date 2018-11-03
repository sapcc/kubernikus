package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

type PyrolisisTests struct {
	Kubernikus *framework.Kubernikus
	OpenStack  *framework.OpenStack
	Reuse      bool
}

func (p *PyrolisisTests) Run(t *testing.T) {
	if p.Reuse == false {
		quota := t.Run("SettingKlustersOnFire", p.SettingKlustersOnFire)
		require.True(t, quota, "Klusters must burn")

		t.Run("Wait", func(t *testing.T) {
			t.Run("Klusters", p.WaitForE2EKlustersTerminated)
		})
	}

	cleanup := t.Run("CleanupBackupStorageContainers", p.CleanupBackupStorageContainers)
	require.True(t, cleanup, "Etcd backup storage container cleanup failed")
}

func (p *PyrolisisTests) SettingKlustersOnFire(t *testing.T) {
	res, err := p.Kubernikus.Client.Operations.ListClusters(
		operations.NewListClustersParams(),
		p.Kubernikus.AuthInfo,
	)
	require.NoError(t, err, "There should be no error while listing klusters")

	for _, kluster := range res.Payload {
		if strings.HasPrefix(kluster.Name, "e2e-") {
			t.Run(fmt.Sprintf("TerminatingKluster-%v", kluster.Name), func(t *testing.T) {
				_, err := p.Kubernikus.Client.Operations.TerminateCluster(
					operations.NewTerminateClusterParams().WithName(kluster.Name),
					p.Kubernikus.AuthInfo,
				)
				assert.NoError(t, err, "There should be no error while terminating klusters")
			})
		}
	}
}

func (p *PyrolisisTests) WaitForE2EKlustersTerminated(t *testing.T) {
	err := p.Kubernikus.WaitForKlusters("e2e-", 0, WaitForKlusterToBeDeletedTimeout)
	assert.NoError(t, err, "E2E Klusters didn't burn down in time")
}

func (p *PyrolisisTests) CleanupBackupStorageContainers(t *testing.T) {
	storageClient, err := openstack.NewObjectStorageV1(p.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "Could not create object storage client")

	containersListOpts := containers.ListOpts{
		Full: false,
	}
	allPages, err := containers.List(storageClient, containersListOpts).AllPages()
	require.NoError(t, err, "There should be no error while listing storage containers")

	allContainers, err := containers.ExtractNames(allPages)
	require.NoError(t, err, "There should be no error while extracting storage containers")

	objectsListOpts := objects.ListOpts{
		Full: false,
	}
	for _, container := range allContainers {
		if strings.HasPrefix(container, etcd_util.BackupStorageContainerBase) {
			allPages, err := objects.List(storageClient, container, objectsListOpts).AllPages()
			require.NoError(t, err, "There should be no error while lising objetcs in container %s", container)

			allObjects, err := objects.ExtractNames(allPages)
			require.NoError(t, err, "There should be no error while extracting objetcs names for container %s", container)

			for _, object := range allObjects {
				_, err := objects.Delete(storageClient, container, object, objects.DeleteOpts{}).Extract()
				require.NoError(t, err, "There should be no error while deleting object %s/%s", container, object)
			}

			_, err = containers.Delete(storageClient, container).Extract()
			require.NoError(t, err, "There should be no error while deleting storage container: %s", container)
		}
	}
}
