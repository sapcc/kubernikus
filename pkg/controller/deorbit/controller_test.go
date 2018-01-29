package deorbit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

var (
	ServerTimeout = errors.NewServerTimeout(core_v1.Resource("services"), "GET", 0)
)

func TestDeborit(testing *testing.T) {
	reconciler := &DeorbitReconciler{}

	deorbiter := &FakeDeorbiter{
		CinderPVCCount: 3,
		LBServiceCount: 2,
	}

	err := reconciler.doDeorbit(deorbiter)
	err = reconciler.doSelfDestruct(deorbiter, err)
	assert.Equal(testing, true, deorbiter.HasCalledDeletePersistentVolumeClaims)
	assert.Equal(testing, true, deorbiter.HasCalledDeleteServices)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForPersistentVolumeCleanup)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForServiceCleanup)
	assert.Equal(testing, false, deorbiter.HasCalledSeldDestruct)
	assert.NoError(testing, err)

	deorbiter = &FakeDeorbiter{
		CinderPVCCount: 0,
		LBServiceCount: 0,
	}

	err = reconciler.doDeorbit(deorbiter)
	err = reconciler.doSelfDestruct(deorbiter, err)
	assert.Equal(testing, true, deorbiter.HasCalledDeletePersistentVolumeClaims)
	assert.Equal(testing, true, deorbiter.HasCalledDeleteServices)
	assert.Equal(testing, false, deorbiter.HasCalledWaitForPersistentVolumeCleanup)
	assert.Equal(testing, false, deorbiter.HasCalledWaitForServiceCleanup)
	assert.Equal(testing, false, deorbiter.HasCalledSeldDestruct)
	assert.NoError(testing, err)

	deorbiter = &FakeDeorbiter{
		CinderPVCCount: 3,
		LBServiceCount: 2,
		APIDown:        true,
	}

	err = reconciler.doDeorbit(deorbiter)
	err = reconciler.doSelfDestruct(deorbiter, ServerTimeout)
	assert.Equal(testing, true, deorbiter.HasCalledDeletePersistentVolumeClaims)
	assert.Equal(testing, true, deorbiter.HasCalledDeleteServices)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForPersistentVolumeCleanup)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForServiceCleanup)
	assert.Equal(testing, true, deorbiter.HasCalledSeldDestruct)
	assert.Equal(testing, APIUnavailable, deorbiter.SelfDestructReason)
	assert.NoError(testing, err)

	deorbiter = &FakeDeorbiter{
		CinderPVCCount: 3,
		LBServiceCount: 2,
		Hanging:        true,
	}

	err = reconciler.doDeorbit(deorbiter)
	err = reconciler.doSelfDestruct(deorbiter, err)
	assert.Equal(testing, true, deorbiter.HasCalledDeletePersistentVolumeClaims)
	assert.Equal(testing, true, deorbiter.HasCalledDeleteServices)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForPersistentVolumeCleanup)
	assert.Equal(testing, true, deorbiter.HasCalledWaitForServiceCleanup)
	assert.Equal(testing, true, deorbiter.HasCalledSeldDestruct)
	assert.Equal(testing, DeorbitHanging, deorbiter.SelfDestructReason)
	assert.NoError(testing, err)
}
