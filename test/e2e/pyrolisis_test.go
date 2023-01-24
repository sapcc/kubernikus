package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/snapshots"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/strings/slices"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	CleanupBackupContainerDeleteInterval       = 1 * time.Second
	CleanupBackupContainerDeleteTimeout        = 1 * time.Minute
	WaitForSnapshotDeletionTimeoutInSeconds    = 300
	WaitForVolumeStabilisationTimeoutInSeconds = 300
)

var StableVolumeStatesForDeletion = [4]string{"available", "error", "error_restoring", "error_managing"}

type PyrolisisTests struct {
	Kubernikus *framework.Kubernikus
	OpenStack  *framework.OpenStack
	Reuse      bool
	Isolate    bool
}

func (p *PyrolisisTests) Run(t *testing.T) {
	if p.Reuse == false && p.Isolate == false {
		quota := t.Run("SettingKlustersOnFire", p.SettingKlustersOnFire)
		require.True(t, quota, "Klusters must burn")

		t.Run("CleanupSnapshots", p.CleanupSnapshots)
		t.Run("CleanupVolumes", p.CleanupVolumes)

		t.Run("Wait", func(t *testing.T) {
			t.Run("Klusters", p.WaitForE2EKlustersTerminated)
		})

		t.Run("CleanupInstances", p.CleanupInstances)
	}

	cleanupStorageContainer := t.Run("CleanupBackupStorageContainers", p.CleanupBackupStorageContainers)
	require.True(t, cleanupStorageContainer, "Etcd backup storage container cleanup failed")

	t.Run("CleanupLoadbalancers", p.CleanupLoadbalancers)
}

func (p *PyrolisisTests) SettingKlustersOnFire(t *testing.T) {
	klusters, err := listKlusters(p.Kubernikus.Client.Operations, p.Kubernikus.AuthInfo)
	require.NoError(t, err, "There should be no error while listing klusters")

	for _, kluster := range klusters {
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

func listKlusters(client operations.ClientService, authinfo runtime.ClientAuthInfoWriter) ([]*models.Kluster, error) {
	res, err := client.ListClusters(
		operations.NewListClustersParams(),
		authinfo,
	)

	if err != nil {
		return nil, err
	}

	return res.Payload, nil
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

	klusters, err := listKlusters(p.Kubernikus.Client.Operations, p.Kubernikus.AuthInfo)
	require.NoError(t, err, "There should be no error while listing klusters")

	// do not delete containers where there is still a kluster running
	var containersToDelete []string
	for _, container := range allContainers {
		found := false

		for _, kluster := range klusters {
			if strings.HasPrefix(container, fmt.Sprintf("%s-%s", etcd_util.BackupStorageContainerBase, kluster.Name)) {
				found = true
			}
		}

		if !found {
			containersToDelete = append(containersToDelete, container)
		}
	}

	objectsListOpts := objects.ListOpts{
		Full: false,
	}

	for _, container := range containersToDelete {
		if strings.HasPrefix(container, etcd_util.BackupStorageContainerBase) {
			allPages, err := objects.List(storageClient, container, objectsListOpts).AllPages()
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				continue
			}
			require.NoError(t, err, "There should be no error while lising objetcs in container %s:", container)

			allObjects, err := objects.ExtractNames(allPages)
			require.NoError(t, err, "There should be no error while extracting objetcs names for container %s", container)

			for _, object := range allObjects {
				_, err := objects.Delete(storageClient, container, object, objects.DeleteOpts{}).Extract()
				//Ignore 404 from swift, this can happen for a successful delete becase of the eventual consistency
				if _, ok := err.(gophercloud.ErrDefault404); ok {
					continue
				}
				require.NoError(t, err, "There should be no error while deleting object %s/%s", container, object)
			}

			err = wait.PollImmediate(CleanupBackupContainerDeleteInterval, CleanupBackupContainerDeleteTimeout,
				func() (bool, error) {
					_, err := containers.Delete(storageClient, container).Extract()
					if _, ok := err.(gophercloud.ErrDefault409); ok {
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

func (p *PyrolisisTests) CleanupSnapshots(t *testing.T) {
	storageClient, err := openstack.NewBlockStorageV3(p.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error while creating storage client")

	project, err := tokens.Get(p.OpenStack.Identity, p.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while extracting the project")

	listOpts := snapshots.ListOpts{
		TenantID: project.ID,
	}

	allPages, err := snapshots.List(storageClient, listOpts).AllPages()
	require.NoError(t, err, "There should be no error retieving all snapshot pages")

	allSnapshots, err := snapshots.ExtractSnapshots(allPages)
	require.NoError(t, err, "There should be no error extracting all snapshots")

	for _, snapshot := range allSnapshots {

		volume, volumeErr := volumes.Get(storageClient, snapshot.VolumeID).Extract()
		require.NoError(t, volumeErr, "There should be no error getting volume of snapshot")

		// only delete snapshot if attached volume was created by our tests
		if volume.Metadata["csi.storage.k8s.io/pvc/name"] != PVCTestName {
			continue
		}

		err := snapshots.Delete(storageClient, snapshot.ID).ExtractErr()
		require.NoError(t, err, "There should be no error deleting the snapshot")

		deleteError := gophercloud.WaitFor(WaitForSnapshotDeletionTimeoutInSeconds,
			func() (bool, error) {
				_, err := snapshots.Get(storageClient, snapshot.ID).Extract()
				if err == nil {
					return false, nil
				}
				return true, nil
			})
		require.NoError(t, deleteError, "There should be no error waiting for the snapshot deletion to finish")
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

		// Make sure volume is in stable state for deletion
		volumeStableErr := gophercloud.WaitFor(WaitForVolumeStabilisationTimeoutInSeconds,
			func() (bool, error) {
				transientVolume, err := volumes.Get(storageClient, vol.ID).Extract()
				if err != nil {
					return false, err
				}
				if slices.Contains(StableVolumeStatesForDeletion[:], transientVolume.Status) {
					return true, nil
				}
				return false, nil
			},
		)

		// Volume has been deleted while waiting for stable state
		if volumeStableErr.Error() == "Resource not found" {
			continue
		}

		require.NoError(t, volumeStableErr, "All volumes with snapshots attached must be in a stable state for deletion")

		// in-tree
		if strings.HasPrefix(vol.Name, "kubernetes-dynamic-pvc-") &&
			strings.HasPrefix(vol.Metadata["kubernetes.io/created-for/pvc/namespace"], "e2e-volumes-") {
			err := volumes.Delete(storageClient, vol.ID, volumes.DeleteOpts{}).ExtractErr()
			require.NoError(t, err, "There should be no error while deleting volume %s (%s)", vol.Name, vol.ID)
		}
		// CSI
		if strings.HasPrefix(vol.Name, "pv-e2e-") &&
			strings.HasPrefix(vol.Metadata["cinder.csi.openstack.org/cluster"], "e2e-") {
			err := volumes.Delete(storageClient, vol.ID, volumes.DeleteOpts{}).ExtractErr()
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

func (p *PyrolisisTests) CleanupLoadbalancers(t *testing.T) {
	lbClient, err := openstack.NewLoadBalancerV2(p.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error getting a loadbalancer client")

	allPages, err := loadbalancers.List(lbClient, loadbalancers.ListOpts{}).AllPages()
	require.NoError(t, err, "There should be no error while listing loadbalancers")

	allLoadbalancers, err := loadbalancers.ExtractLoadBalancers(allPages)
	require.NoError(t, err, "There should be no error while extracting loadbalancers")

	klusters, err := listKlusters(p.Kubernikus.Client.Operations, p.Kubernikus.AuthInfo)
	require.NoError(t, err, "There should be no error while listing klusters")

	// do not delete loadbalancers where there is still a kluster running
	var lbsToDelete []loadbalancers.LoadBalancer
	for _, lb := range allLoadbalancers {
		found := false

		for _, kluster := range klusters {
			if strings.HasPrefix(lb.Name, fmt.Sprintf("kube_service_%s", kluster.Name)) {
				found = true
			}
		}

		if !found {
			lbsToDelete = append(lbsToDelete, lb)
		}
	}

	for _, lb := range lbsToDelete {
		if strings.HasSuffix(lb.Name, "_e2e-lb") {
			err := loadbalancers.Delete(lbClient, lb.ID, loadbalancers.DeleteOpts{Cascade: true}).ExtractErr()

			// Ignore PENDING_DELETE error
			if _, ok := err.(gophercloud.ErrDefault409); ok {
				continue
			}

			require.NoError(t, err, "There should be no error while deleting loadbalancer %s", lb.Name)
		}
	}
}
