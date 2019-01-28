package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	CleanupBackupContainerDeleteInterval = 1 * time.Second
	CleanupBackupContainerDeleteTimeout  = 1 * time.Minute
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

		cleanupStorageContainer := t.Run("CleanupBackupStorageContainers", p.CleanupBackupStorageContainers)
		require.True(t, cleanupStorageContainer, "Etcd backup storage container cleanup failed")

		t.Run("CleanupVolumes", p.CleanupVolumes)
		t.Run("CleanupInstances", p.CleanupInstances)
	}
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

			err = wait.PollImmediate(CleanupBackupContainerDeleteInterval, CleanupBackupContainerDeleteTimeout,
				func() (bool, error) {
					_, err := containers.Delete(storageClient, container).Extract()
					if errResponseCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok && errResponseCode.Actual == 409 {
						return false, nil
					}
					//Ignore 404 from swift, this can happen for a successful delete becase of the eventual consistency
					if _, ok := err.(gophercloud.ErrDefault404); ok {
						return true, nil
					}
					return true, err
				})
			require.NoError(t, err, "There should be no error while deleting storage container: %s", container)
		}
	}
}

func (p *PyrolisisTests) CleanupVolumes(t *testing.T) {
	storageClient, err := openstack.NewBlockStorageV3(p.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "Could not create block storage client")

	project, err := tokens.Get(p.OpenStack.Identity, p.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while extracting the project")

	volumeListOpts := volumes.ListOpts{
		TenantID: project.ID,
	}

	allPages, err := volumes.List(storageClient, volumeListOpts).AllPages()
	require.NoError(t, err, "There should be no error while retrieving volume pages")

	allVolumes, err := volumes.ExtractVolumes(allPages)
	require.NoError(t, err, "There should be no error while extracting volumes")

	for _, vol := range allVolumes {
		if strings.HasPrefix(vol.Name, "kubernetes-dynamic-pvc-") &&
			strings.HasPrefix(vol.Metadata["kubernetes.io/created-for/pvc/namespace"], "e2e-volumes-") {
			err := volumes.Delete(storageClient, vol.ID).ExtractErr()
			require.NoError(t, err, "There should be no error while deleting volume %s (%s)", vol.Name, vol.ID)
		}
	}
}

func (p *PyrolisisTests) CleanupInstances(t *testing.T) {
	computeClient, err := openstack.NewComputeV2(p.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error creating compute client")

	project, err := tokens.Get(p.OpenStack.Identity, p.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while extracting the project")

	serversListOpts := servers.ListOpts{
		Name:     "e2e-",
		TenantID: project.ID,
	}

	allPages, err := servers.List(computeClient, serversListOpts).AllPages()
	require.NoError(t, err, "There should be no error while listing all servers")

	allServers, err := servers.ExtractServers(allPages)
	require.NoError(t, err, "There should be no error while extracting all servers")

	for _, srv := range allServers {
		err := servers.Delete(computeClient, srv.ID).ExtractErr()
		require.NoError(t, err, "There should be no error while deleting server %s (%s)", srv.Name, srv.ID)
	}
}
