package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	blockstorage_quota "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	compute_quota "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	KlusterPhaseBecomesTerminatingTimeout = 1 * time.Minute
	WaitForKlusterToBeDeletedTimeout      = 10 * time.Minute
)

type CleanupTests struct {
	Kubernikus  *framework.Kubernikus
	OpenStack   *framework.OpenStack
	KlusterName string
	Reuse       bool
}

func (s *CleanupTests) Run(t *testing.T) {
	if t.Run("Cluster/Terminate", s.TerminateCluster) {
		t.Run("Cluster/BecomesTerminating", s.KlusterPhaseBecomesTerminating)
		t.Run("Cluster/IsDeleted", s.WaitForKlusterToBeDeleted)

		if s.Reuse == false {
			t.Run("QuotaPostFlightCheck", s.QuotaPostFlightCheck)
			t.Run("ServerGroupsGotDeleted", s.ServerGroupsGotDeleted)
		}
	}
}

func (s *CleanupTests) TerminateCluster(t *testing.T) {
	_, err := s.Kubernikus.Client.Operations.TerminateCluster(
		operations.NewTerminateClusterParams().WithName(s.KlusterName),
		s.Kubernikus.AuthInfo,
	)
	assert.NoError(t, err, "There should be no error")
}

func (s *CleanupTests) WaitForKlusterToBeDeleted(t *testing.T) {
	err := s.Kubernikus.WaitForKlusterToBeDeleted(s.KlusterName, WaitForKlusterToBeDeletedTimeout)
	require.NoError(t, err, "There should be no error while waiting %v for the kluster to be deleted", WaitForKlusterToBeDeletedTimeout)
}

func (s *CleanupTests) KlusterPhaseBecomesTerminating(t *testing.T) {
	phase, err := s.Kubernikus.WaitForKlusterPhase(s.KlusterName, models.KlusterPhaseTerminating, KlusterPhaseBecomesTerminatingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhaseTerminating, "Kluster should become Terminating")
	}
}

func (s *CleanupTests) QuotaPostFlightCheck(t *testing.T) {
	project, err := tokens.Get(s.OpenStack.Identity, s.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while getting project from token")
	require.NotNil(t, project, "project returned from Token %s was nil. WTF?", s.OpenStack.Provider.Token())

	quota, err := compute_quota.GetDetail(s.OpenStack.Compute, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting compute quota details")

	storage, err := blockstorage_quota.GetUsage(s.OpenStack.BlockStorage, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting storage quota details")

	assert.Zero(t, quota.Cores.InUse, "There should be no cores left in use")
	assert.Zero(t, quota.Instances.InUse, "There should be no instances left in use")
	assert.Zero(t, quota.RAM.InUse, "There should be no RAM left in use")
	assert.Zero(t, storage.Volumes.InUse, "There should be no Volume left in use")
	assert.Zero(t, storage.Gigabytes.InUse, "There should be no Storage left in use")
}

func (s *CleanupTests) ServerGroupsGotDeleted(t *testing.T) {
	computeClient, err := openstack.NewComputeV2(s.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error creating compute client")

	allPages, err := servergroups.List(computeClient).AllPages()
	require.NoError(t, err, "There should be no error listing server groups")

	allGroups, err := servergroups.ExtractServerGroups(allPages)
	require.NoError(t, err, "There should be no error extracting server groups")

	count := 0
	for _, sg := range allGroups {
		if strings.HasPrefix(sg.Name, "e2e-") {
			count++
		}
	}
	require.Equal(t, 0, count, "There should be no server groups left")
}
