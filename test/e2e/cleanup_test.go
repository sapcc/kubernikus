package main

import (
	"testing"
	"time"

	blockstorage_quota "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	compute_quota "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
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

	quota, err := compute_quota.GetDetail(s.OpenStack.Compute, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting compute quota details")

	storage, err := blockstorage_quota.GetUsage(s.OpenStack.BlockStorage, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting storage quota details")

	assert.True(t, quota.Cores.InUse == 0, "There should be no cores left in use")
	assert.True(t, quota.Instances.InUse == 0, "There should be no instances left in use")
	assert.True(t, quota.RAM.InUse == 0, "There should be no RAM left in use")
	assert.True(t, storage.Volumes.InUse == 0, "There should be no Volume left in use")
	assert.True(t, storage.Gigabytes.InUse == 0, "There should be no Storage left in use")
}
