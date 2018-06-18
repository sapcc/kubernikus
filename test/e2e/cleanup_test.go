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
	KlusterPhaseBecomesTerminatingTimeout = 1 * time.Minute
	WaitForKlusterToBeDeletedTimeout      = 5 * time.Minute
)

type CleanupTests struct {
	Kubernikus  *framework.Kubernikus
	KlusterName string
}

func (s *CleanupTests) Run(t *testing.T) {
	if t.Run("Cluster/Terminate", s.TerminateCluster) {
		t.Run("Cluster/BecomesTerminating", s.KlusterPhaseBecomesTerminating)
		t.Run("Cluster/IsDeleted", s.WaitForKlusterToBeDeleted)
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
