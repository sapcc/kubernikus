package util

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Masterminds/semver"
	api_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	clientset "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
	listers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
	utils_pod "github.com/sapcc/kubernikus/pkg/util/pod"
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
		context.TODO(),
		kluster.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"status":{"migrationsPending":%s}}`, strconv.FormatBool(pending))),
		meta_v1.PatchOptions{},
	)
	return err
}

func UpdateKlusterPhase(client clientset.KubernikusV1Interface, kluster *v1.Kluster, phase models.KlusterPhase) error {

	//Because we specify raw json for the patch I put this here to make sure field is still part of the struct
	var _ = kluster.Status.Phase

	//According to this comment https://github.com/kubernetes/kubernetes/issues/21479#issuecomment-186454413
	//patches are retried on the server side so a single try should be sufficient

	_, err := client.Klusters(kluster.Namespace).Patch(
		context.TODO(),
		kluster.Name,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"status":{"phase":"%s"}}`, phase)),
		meta_v1.PatchOptions{},
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
	apiSecret, err := client.CoreV1().Secrets(kluster.Namespace).Create(context.TODO(), &s, meta_v1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return KlusterSecret(client, kluster)
	}
	if err != nil {
		return nil, err
	}
	return v1.NewSecret(apiSecret)
}

// NOTE: this is not threadsafe (but we are only calling this once per kluster for the time being)
func UpdateKlusterSecret(client kubernetes.Interface, kluster *v1.Kluster, secret *v1.Secret) error {
	api_secret, err := client.CoreV1().Secrets(kluster.Namespace).Get(context.TODO(), klusterSecretName(kluster), meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	api_secret.Data, err = secret.ToData()
	if err != nil {
		return fmt.Errorf("Failed to serialize secret data: %s", err)
	}
	_, err = client.CoreV1().Secrets(kluster.Namespace).Update(context.TODO(), api_secret, meta_v1.UpdateOptions{})
	return err

}

func DeleteKlusterSecret(client kubernetes.Interface, kluster *v1.Kluster) error {
	err := client.CoreV1().Secrets(kluster.Namespace).Delete(context.TODO(), klusterSecretName(kluster), meta_v1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func klusterSecretName(k *v1.Kluster) string {
	return k.Name + "-secret"
}

func KlusterSecret(client kubernetes.Interface, kluster *v1.Kluster) (*v1.Secret, error) {
	secret, err := client.CoreV1().Secrets(kluster.Namespace).Get(context.TODO(), klusterSecretName(kluster), meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return v1.NewSecret(secret)
}

func KlusterVersionConstraint(kluster *v1.Kluster, constraint string) (bool, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	v, err := semver.NewVersion(kluster.Spec.Version)
	if err != nil {
		return false, err
	}
	return c.Check(v), nil
}

func KlusterNeedsUpgrade(kluster *v1.Kluster) (bool, error) {

	// from, err := semver.NewVersion(kluster.Status.ApiserverVersion)
	// if err != nil {
	// 	return false, err
	// }

	// to, err := semver.NewVersion(kluster.Spec.Version)
	// if err != nil {
	// 	return false, err
	// }
	// return from.Compare(to) != 0 && (from.Minor() == to.Minor() || from.Minor()+1 == to.Minor()), nil
	return false, nil
}

func KlusterPodsReadyCount(kluster *v1.Kluster, podLister listers_core_v1.PodLister) (int, int, error) {
	pods, err := podLister.List(labels.SelectorFromValidatedSet(map[string]string{"release": kluster.GetName()}))
	if err != nil {
		return 0, 0, err
	}
	podsReady := 0
	for _, pod := range pods {
		if utils_pod.IsPodReady(pod) {
			podsReady++
		}
	}
	return podsReady, len(pods), nil

}
