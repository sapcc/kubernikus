package handlers

import (
	"fmt"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/validate"
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCreateCluster(rt *api.Runtime) operations.CreateClusterHandler {
	return &createCluster{rt}
}

type createCluster struct {
	*api.Runtime
}

func (d *createCluster) Handle(params operations.CreateClusterParams, principal *models.Principal) middleware.Responder {
	name := *params.Body.Name

	if err := validate.UniqueItems("name", "body", params.Body.Spec.NodePools); err != nil {
		return NewErrorResponse(&operations.CreateClusterDefault{}, int(err.Code()), err.Error())
	}

	var nodePools []v1.NodePool
	var nodePoolInfos []v1.NodePoolInfo
	if params.Body.Spec != nil && params.Body.Spec.NodePools != nil {
		nodePools = []v1.NodePool{}
		nodePoolInfos = []v1.NodePoolInfo{}
		for _, pPool := range params.Body.Spec.NodePools {
			nodePools = append(nodePools, v1.NodePool{
				Name:   *pPool.Name,
				Size:   int(*pPool.Size),
				Flavor: *pPool.Flavor,
				Image:  pPool.Image,
			})

			nodePoolInfos = append(nodePoolInfos, v1.NodePoolInfo{
				Name:        *pPool.Name,
				Size:        int(*pPool.Size),
				Running:     0,
				Healthy:     0,
				Schedulable: 0,
			})
		}
	}

	kluster := &v1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", name, principal.Account),
			Labels:      map[string]string{"account": principal.Account},
			Annotations: map[string]string{"creator": principal.Name},
		},
		Spec: v1.KlusterSpec{
			Name:      name,
			NodePools: nodePools,
		},
		Status: v1.KlusterStatus{
			Kluster: v1.KlusterInfo{
				State: v1.KlusterPending,
			},
		},
	}

	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Create(kluster)
	if err != nil {
		glog.Errorf("Failed to create cluster: %s", err)
		if apierrors.IsAlreadyExists(err) {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Cluster with name %s already exists", name)
		}
		return NewErrorResponse(&operations.CreateClusterDefault{}, 500, err.Error())
	}

	return operations.NewCreateClusterCreated().WithPayload(clusterModelFromTPR(kluster))
}
