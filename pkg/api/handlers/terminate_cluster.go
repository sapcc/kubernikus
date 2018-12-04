package handlers

import (
	"github.com/go-openapi/runtime/middleware"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func NewTerminateCluster(rt *api.Runtime) operations.TerminateClusterHandler {
	return &terminateCluster{rt}
}

type terminateCluster struct {
	*api.Runtime
}

func (d *terminateCluster) Handle(params operations.TerminateClusterParams, principal *models.Principal) middleware.Responder {

	kluster := d.Kubernikus.Kubernikus().Klusters(d.Namespace)
	_, err := kluster.Get(qualifiedName(params.Name, principal.Account), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.TerminateClusterDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.TerminateClusterDefault{}, 500, err.Error())
	}

	_, err = editCluster(kluster, principal, params.Name, func(kluster *v1.Kluster) error {
		kluster.Status.Phase = models.KlusterPhaseTerminating
		kluster.Status.Message = "Cluster terminating"

		return nil
	})
	if err != nil {
		return NewErrorResponse(&operations.TerminateClusterDefault{}, 500, err.Error())
	}

	// This issues a delete request for the Kluster CRD
	//
	// It actually adds a `metadata.DeletionTimestamp` to the Kluster. The Garbage-
	// Controller will pick up on that and delete the resource. But only when the
	// metadata.Finalizers array is empty. Until then the Kluster will keep on
	// existing.
	//
	// Kubernikus Controllers are required to add/remove Finalizers if clean-up is
	// required once a Kluster is deleted.
	if err := kluster.Delete(qualifiedName(params.Name, principal.Account), &metav1.DeleteOptions{}); err != nil {
		return NewErrorResponse(&operations.TerminateClusterDefault{}, 500, err.Error())
	}

	return operations.NewTerminateClusterAccepted()
}
