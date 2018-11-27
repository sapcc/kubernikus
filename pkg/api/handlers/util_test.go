package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

func TestDetectNodePoolChanges(t *testing.T) {

	np := models.NodePool{Name: "pool", Size: 0, Image: "image", Flavor: "flavor"}
	npScaled := models.NodePool{Name: "pool", Size: 5, Image: "image", Flavor: "flavor"}
	npChanged := models.NodePool{Name: "pool", Size: 0, Image: "image:v2", Flavor: "flavor"}
	npNew := models.NodePool{Name: "pool_new", Size: 0, Image: "image", Flavor: "flavor"}

	nodePoolListOriginal := []models.NodePool{np, npNew}
	nodePoolListScaled := []models.NodePool{npScaled, npNew}
	nodePoolListRemoved := []models.NodePool{np}
	nodePoolListChanged := []models.NodePool{npChanged}

	deleteList, err := detectNodePoolChanges(nodePoolListOriginal, nodePoolListScaled)
	assert.Len(t, deleteList, 0)
	assert.Nil(t, err)

	deleteList, err = detectNodePoolChanges(nodePoolListOriginal, nodePoolListRemoved)
	assert.Len(t, deleteList, 1)
	assert.Nil(t, err)

	deleteList, err = detectNodePoolChanges(nodePoolListOriginal, nodePoolListChanged)
	assert.Len(t, deleteList, 0)
	assert.NotNil(t, err)

}

func TestNodePoolEqualsWithScaling(t *testing.T) {

	np := models.NodePool{Name: "pool", Size: 0, Image: "image", Flavor: "flavor"}
	npScaled := models.NodePool{Name: "pool", Size: 5, Image: "image", Flavor: "flavor"}
	npChanged := models.NodePool{Name: "pool", Size: 0, Image: "image:v2", Flavor: "flavor"}

	assert.Nil(t, nodePoolEqualsWithScaling(np, npScaled))
	assert.NotNil(t, nodePoolEqualsWithScaling(np, npChanged))

}
