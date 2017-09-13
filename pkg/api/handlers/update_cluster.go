package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func NewUpdateCluster(rt *api.Runtime) operations.UpdateClusterHandler {
	return &updateCluster{rt}
}

type updateCluster struct {
	*api.Runtime
}

func (d *updateCluster) Handle(params operations.UpdateClusterParams, principal *models.Principal) middleware.Responder {

	kluster, err := editCluster(d.Kubernikus.Kubernikus().Klusters(d.Namespace), principal, params.Name, func(kluster *v1.Kluster) {
		// Update Sizes
		for _, pPool := range params.Body.Spec.NodePools {
			isNewPool := true

			for _, kPool := range kluster.Spec.NodePools {
				if pPool.Name == kPool.Name {
					kPool.Size = int(pPool.Size)
					isNewPool = false
				}
			}

			if isNewPool {
				kluster.Spec.NodePools = append(kluster.Spec.NodePools, v1.NodePool{
					Name:   pPool.Name,
					Size:   int(pPool.Size),
					Flavor: pPool.Flavor,
					Image:  pPool.Image,
				})
			}
		}

		for i, kPool := range kluster.Spec.NodePools {
			isDeleted := true
			for _, pPool := range params.Body.Spec.NodePools {
				if pPool.Name == kPool.Name {
					isDeleted = false
					break
				}
			}
			if isDeleted {
				// wtf? I want my ruby back...
				kluster.Spec.NodePools[i] = kluster.Spec.NodePools[len(kluster.Spec.NodePools)-1]
				kluster.Spec.NodePools = kluster.Spec.NodePools[:len(kluster.Spec.NodePools)-1]
			}
		}

	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.UpdateClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
	}
	return operations.NewUpdateClusterOK().WithPayload(clusterModelFromTPR(kluster))
}
