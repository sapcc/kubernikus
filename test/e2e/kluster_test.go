package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	KlusterPhaseBecomesPendingTimeout     = 1 * time.Minute
	KlusterPhaseBecomesCreatingTimeout    = 1 * time.Minute
	KlusterPhaseBecomesRunningTimeout     = 5 * time.Minute
	KlusterPhaseBecomesTerminatingTimeout = 1 * time.Minute
	KlusterFinishedTerminationTermination = 5 * time.Minute
)

type KlusterTests struct {
	Kubernikus  *framework.Kubernikus
	KlusterName string
}

func (k *KlusterTests) KlusterPhaseBecomesPending(t *testing.T) {
	t.Parallel()

	phase, err := k.Kubernikus.WaitForKlusterPhase(k.KlusterName, models.KlusterPhasePending, KlusterPhaseBecomesPendingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhasePending, "Kluster should become Pending")
	}
}

func (k *KlusterTests) KlusterPhaseBecomesCreating(t *testing.T) {
	t.Parallel()

	phase, err := k.Kubernikus.WaitForKlusterPhase(k.KlusterName, models.KlusterPhaseCreating, KlusterPhaseBecomesCreatingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhaseCreating, "Kluster should become Creating")
	}
}

func (k *KlusterTests) KlusterPhaseBecomesRunning(t *testing.T) {
	t.Parallel()

	phase, err := k.Kubernikus.WaitForKlusterPhase(k.KlusterName, models.KlusterPhaseRunning, KlusterPhaseBecomesRunningTimeout)

	if assert.NoError(t, err, "There should be no error") {
		require.Equal(t, phase, models.KlusterPhaseRunning, "Kluster should become Running")
	}
}

func (k *KlusterTests) KlusterPhaseBecomesTerminating(t *testing.T) {
	t.Parallel()

	phase, err := k.Kubernikus.WaitForKlusterPhase(k.KlusterName, models.KlusterPhaseTerminating, KlusterPhaseBecomesTerminatingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhaseTerminating, "Kluster should become Terminating")
	}
}
