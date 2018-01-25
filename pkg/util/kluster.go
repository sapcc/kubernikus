package util

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	clientset "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
	listers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
)

func EnsureFinalizerCreated(client clientset.KubernikusV1Interface, lister listers_kubernikus.KlusterLister, kluster *v1.Kluster, finalizer string) (err error) {
	if kluster.NeedsFinalizer(finalizer) {
		_, err = UpdateKlusterWithRetries(client.Klusters(kluster.Namespace), lister.Klusters(kluster.Namespace), kluster.Name, func(kluster *v1.Kluster) error {
			kluster.AddFinalizer(finalizer)
			return nil
		})
	}
	return err
}

func EnsureFinalizerRemoved(client clientset.KubernikusV1Interface, lister listers_kubernikus.KlusterLister, kluster *v1.Kluster, finalizer string) (err error) {
	if kluster.HasFinalizer(finalizer) {
		_, err = UpdateKlusterWithRetries(client.Klusters(kluster.Namespace), lister.Klusters(kluster.Namespace), kluster.Name, func(kluster *v1.Kluster) error {
			kluster.RemoveFinalizer(finalizer)
			return nil
		})
	}
	return err
}

func UpdateKlusterMigrationStatus(client clientset.KubernikusV1Interface, kluster *v1.Kluster, pending bool) error {

	if kluster.Status.MigrationsPending == pending {
		return nil // already up to date
	}

	//According to this comment https://github.com/kubernetes/kubernetes/issues/21479#issuecomment-186454413
	//patches are retried on the server side so a single try should be sufficient

	_, err := client.Klusters(kluster.Namespace).Patch(
		kluster.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"status":{"migrationsPending":%s}}`, strconv.FormatBool(pending))),
	)
	return err
}

func UpdateKlusterPhase(client clientset.KubernikusV1Interface, kluster *v1.Kluster, phase models.KlusterPhase) error {

	//Because we specify raw json for the patch I put this here to make sure field is still part of the struct
	var _ = kluster.Status.Phase

	//According to this comment https://github.com/kubernetes/kubernetes/issues/21479#issuecomment-186454413
	//patches are retried on the server side so a single try should be sufficient

	_, err := client.Klusters(kluster.Namespace).Patch(
		kluster.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"status":{"phase":"%s"}}`, phase)),
	)
	return err
}
