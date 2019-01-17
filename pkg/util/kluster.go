package util

import (
	"fmt"
	"strconv"

	api_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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

func EnsureKlusterSecret(client kubernetes.Interface, kluster *v1.Kluster) (*v1.Secret, error) {

	klusterRef := NewOwnerRef(kluster, v1.SchemeGroupVersion.WithKind("Kluster"))
	s := api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            klusterSecretName(kluster),
			Labels:          kluster.Labels,
			OwnerReferences: []meta_v1.OwnerReference{*klusterRef},
		},
	}
	apiSecret, err := client.Core().Secrets(kluster.Namespace).Create(&s)
	if apierrors.IsAlreadyExists(err) {
		return KlusterSecret(client, kluster)
	}
	if err != nil {
		return nil, err
	}
	return v1.NewSecret(apiSecret)
}

// NOTE: this is not threadsafe (but we are only calling this once per kluster for the time beeing)
func UpdateKlusterSecret(client kubernetes.Interface, kluster *v1.Kluster, secret *v1.Secret) error {
	api_secret, err := client.Core().Secrets(kluster.Namespace).Get(klusterSecretName(kluster), meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	api_secret.Data, err = secret.ToData()
	if err != nil {
		return fmt.Errorf("Failed to serialize secret data: %s", err)
	}
	_, err = client.Core().Secrets(kluster.Namespace).Update(api_secret)
	return err

}

func DeleteKlusterSecret(client kubernetes.Interface, kluster *v1.Kluster) error {
	err := client.Core().Secrets(kluster.Namespace).Delete(klusterSecretName(kluster), nil)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func klusterSecretName(k *v1.Kluster) string {
	return k.Name + "-secret"
}

func KlusterSecret(client kubernetes.Interface, kluster *v1.Kluster) (*v1.Secret, error) {
	secret, err := client.CoreV1().Secrets(kluster.Namespace).Get(klusterSecretName(kluster), meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return v1.NewSecret(secret)
}
