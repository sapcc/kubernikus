package handlers

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
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

	kluster, err := editCluster(d.Kubernikus.KubernikusV1().Klusters(d.Namespace), principal, params.Name, func(kluster *v1.Kluster) error {
		// ensure audit value reaches the spec so it
		// can be considered when upgrading the kluster
		kluster.Spec.Audit = params.Body.Spec.Audit

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
				if specPool.Name != paramPool.Name {
					continue
				}

				nodePools[i].AvailabilityZone = specPool.AvailabilityZone

				if paramPool.CgroupsV1 == nil {
					nodePools[i].CgroupsV1 = specPool.CgroupsV1
				}

				if paramPool.Config == nil {
					nodePools[i].Config = specPool.Config
				} else {
					if paramPool.Config.AllowReboot == nil {
						nodePools[i].Config.AllowReboot = specPool.Config.AllowReboot
					}

					if paramPool.Config.AllowReplace == nil {
						nodePools[i].Config.AllowReplace = specPool.Config.AllowReplace
					}
				}
			}
		}

		// restore defaults
		for i, paramPool := range nodePools {

			allowReboot := true
			allowReplace := true
			if paramPool.Config == nil {
				nodePools[i].Config = &models.NodePoolConfig{
					AllowReboot:  &allowReboot,
					AllowReplace: &allowReplace,
				}
			}

			if nodePools[i].Config.AllowReboot == nil {
				nodePools[i].Config.AllowReboot = &allowReboot
			}

			if nodePools[i].Config.AllowReplace == nil {
				nodePools[i].Config.AllowReplace = &allowReplace
			}

		}

		// Update nodepool
		kluster.Spec.NodePools = nodePools
		kluster.Spec.SSHPublicKey = params.Body.Spec.SSHPublicKey

		if params.Body.Spec.Openstack.SecurityGroupName != "" {
			kluster.Spec.Openstack.SecurityGroupName = params.Body.Spec.Openstack.SecurityGroupName
		}

		if params.Body.Spec.Version != "" && params.Body.Spec.Version != kluster.Status.ApiserverVersion {
			newVersion, err := semver.NewVersion(params.Body.Spec.Version)
			if err != nil {
				return apierrors.NewBadRequest(fmt.Sprintf("Invalid version (%s) specified for kluster: %s", params.Body.Spec.Version, err))
			}
			currentVersion, err := semver.NewVersion(kluster.Status.ApiserverVersion)
			if err != nil {
				return apierrors.NewInternalError(fmt.Errorf("Can't parse current apiserver version (%s): %s", kluster.Status.ApiserverVersion, err))
			}
			if newVersion.Major() != currentVersion.Major() || newVersion.Minor() < currentVersion.Minor() || newVersion.Minor() > currentVersion.Minor()+1 {
				return apierrors.NewBadRequest(fmt.Sprintf("Can't upgrade from version %s to %s", kluster.Status.ApiserverVersion, params.Body.Spec.Version))
			}
			if kluster.Status.Phase != models.KlusterPhaseRunning {
				return apierrors.NewBadRequest(fmt.Sprintf("Version can be changed in state %s only", models.KlusterPhaseRunning))
			}
			kluster.Spec.Version = params.Body.Spec.Version

			// Update existing nodepools to use flatcar image
			for i, specPool := range kluster.Spec.NodePools {
				if specPool.Image == "coreos-stable-amd64" {
					kluster.Spec.NodePools[i].Image = DEFAULT_IMAGE
				}
			}
		}

		dexEnabled := swag.BoolValue(kluster.Spec.Dex)
		dashboardEnabled := swag.BoolValue(kluster.Spec.Dashboard)
		if params.Body.Spec.Dex != nil {
			dexEnabled = swag.BoolValue(params.Body.Spec.Dex)
		}
		if params.Body.Spec.Dashboard != nil {
			dashboardEnabled = swag.BoolValue(params.Body.Spec.Dashboard)
		}

		if !dexEnabled && dashboardEnabled {
			return apierrors.NewBadRequest(fmt.Sprintf("Dashboard cannot be enabled while Dex is disabled"))
		}

		//Dex value changed
		if params.Body.Spec.Dex != nil && swag.BoolValue(params.Body.Spec.Dex) != swag.BoolValue(kluster.Spec.Dex) {
			kluster.Spec.Dex = params.Body.Spec.Dex
		}

		//Dashboard value changed
		if params.Body.Spec.Dashboard != nil && swag.BoolValue(params.Body.Spec.Dashboard) != swag.BoolValue(kluster.Spec.Dashboard) {

			kluster.Spec.Dashboard = params.Body.Spec.Dashboard
			if dashboardEnabled && kluster.Status.Apiserver != "" {
				apiURL := kluster.Status.Apiserver
				kluster.Status.Dashboard = strings.Replace(apiURL, kluster.GetName(), fmt.Sprintf("dashboard-%s.ingress", kluster.GetName()), -1)
			}

		}

		return nil
	})

	if err != nil {

		switch e := err.(type) {
		case apierrors.APIStatus:
			return NewErrorResponse(&operations.UpdateClusterDefault{}, int(e.Status().Code), err.Error())
		default:
			return NewErrorResponse(&operations.UpdateClusterDefault{}, 500, err.Error())
		}

	}

	return operations.NewUpdateClusterOK().WithPayload(klusterFromCRD(kluster))
}
