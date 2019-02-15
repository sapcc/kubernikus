package handlers

import (
	"fmt"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func NewUpdateCluster(rt *api.Runtime) operations.UpdateClusterHandler {
	return &updateCluster{rt}
}

type updateCluster struct {
	*api.Runtime
}

func (d *updateCluster) Handle(params operations.UpdateClusterParams, principal *models.Principal) middleware.Responder {
	metadata, err := FetchOpenstackMetadataFunc(params.HTTPRequest, principal)
	if err != nil {
		return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
	}

	defaultAVZ, err := getDefaultAvailabilityZone(metadata)
	if err != nil {
		return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
	}

	kluster, err := editCluster(d.Kubernikus.Kubernikus().Klusters(d.Namespace), principal, params.Name, func(kluster *v1.Kluster) error {
		// find the deleted nodepools
		deletedNodePoolNames, err := detectNodePoolChanges(kluster.Spec.NodePools, params.Body.Spec.NodePools)
		if err != nil {
			return err
		}

		// clear the status for the deleted nodepools
		if len(deletedNodePoolNames) > 0 {
			nodePoolInfo := kluster.Status.NodePools
			for _, name := range deletedNodePoolNames {
				for i, statusNodePool := range nodePoolInfo {
					if name == statusNodePool.Name {
						nodePoolInfo = append(nodePoolInfo[:i], nodePoolInfo[i+1:]...)
					}

				}
			}
			kluster.Status.NodePools = nodePoolInfo
		}

		nodePools := params.Body.Spec.NodePools
		//set default image
		for i, pool := range nodePools {
			if pool.Image == "" {
				nodePools[i].Image = DEFAULT_IMAGE
			}
		}

		// Keep previous AVZ
		for _, specPool := range kluster.Spec.NodePools {
			for i, paramPool := range nodePools {
				if specPool.Name == paramPool.Name {
					nodePools[i].AvailabilityZone = specPool.AvailabilityZone
				}
			}
		}

		for i, paramPool := range nodePools {
			// Set default AvailabilityZone
			if paramPool.AvailabilityZone == "" {
				nodePools[i].AvailabilityZone = defaultAVZ
			}

			if err := validateAavailabilityZone(nodePools[i].AvailabilityZone, metadata); err != nil {
				return fmt.Errorf("Availability Zone %s is invalid: %s", nodePools[i].AvailabilityZone, err)
			}
		}

		// Update nodepool
		kluster.Spec.NodePools = nodePools
		kluster.Spec.SSHPublicKey = params.Body.Spec.SSHPublicKey

		if params.Body.Spec.Openstack.SecurityGroupName != "" {
			kluster.Spec.Openstack.SecurityGroupName = params.Body.Spec.Openstack.SecurityGroupName
		}

		return nil
	})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.UpdateClusterDefault{}, 404, "Not found")
		}

		if strings.HasPrefix(err.Error(), "Availability Zone") {
			return NewErrorResponse(&operations.UpdateClusterDefault{}, 409, err.Error())
		}

		return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
	}

	return operations.NewUpdateClusterOK().WithPayload(klusterFromCRD(kluster))
}
