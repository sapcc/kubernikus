package migration

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
)

const NAMESPACE = "test"

func TestInitialMigration(t *testing.T) {

	kluster := &v1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NAMESPACE,
			Name:      "test",
		},
	}
	cs := fake.NewSimpleClientset()
	kcs := kubernikusfake.NewSimpleClientset(kluster)

	var registry Registry
	registry.AddMigration(func(_ []byte, kluster *v1.Kluster, _ kubernetes.Interface) error {
		kluster.Spec.Name = "executed"
		return nil
	})

	if assert.NoError(t, registry.Migrate(kluster, cs, kcs)) {
		kluster, _ = kcs.Kubernikus().Klusters(NAMESPACE).Get("test", metav1.GetOptions{})
		assert.Equal(t, 1, int(kluster.Status.SpecVersion))
		assert.Equal(t, "executed", kluster.Spec.Name)
	}

}

func TestMigration(t *testing.T) {
	kluster := &v1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NAMESPACE,
			Name:      "test",
		},
		Status: models.KlusterStatus{
			SpecVersion: 1,
		},
	}
	cs := fake.NewSimpleClientset()
	kcs := kubernikusfake.NewSimpleClientset(kluster)

	var registry Registry
	registry.AddMigration(func(_ []byte, kluster *v1.Kluster, _ kubernetes.Interface) error {
		t.Error("First migration should be skipped")
		return nil
	})

	registry.AddMigration(func(_ []byte, kluster *v1.Kluster, _ kubernetes.Interface) error {
		kluster.Spec.Name = kluster.Spec.Name + "2"
		return nil
	})

	registry.AddMigration(func(_ []byte, kluster *v1.Kluster, _ kubernetes.Interface) error {
		kluster.Spec.Name = kluster.Spec.Name + "3"
		return nil
	})

	if assert.NoError(t, registry.Migrate(kluster, cs, kcs)) {
		kluster, _ = kcs.Kubernikus().Klusters(NAMESPACE).Get("test", metav1.GetOptions{})
		assert.Equal(t, 3, int(kluster.Status.SpecVersion))
		assert.Equal(t, "23", kluster.Spec.Name)
	}
}

func TestMigrationError(t *testing.T) {
	kluster := &v1.Kluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NAMESPACE,
			Name:      "test",
		},
		Spec: models.KlusterSpec{
			Name: "Before",
		},
	}
	cs := fake.NewSimpleClientset()
	kcs := kubernikusfake.NewSimpleClientset(kluster)

	var registry Registry
	registry.AddMigration(func(_ []byte, kluster *v1.Kluster, _ kubernetes.Interface) error {
		kluster.Spec.Name = "After"
		return errors.New("migration failed")
	})

	if assert.Error(t, registry.Migrate(kluster, cs, kcs)) {
		kluster, _ = kcs.Kubernikus().Klusters(NAMESPACE).Get("test", metav1.GetOptions{})
		assert.Equal(t, 0, int(kluster.Status.SpecVersion))
		assert.Equal(t, "Before", kluster.Spec.Name)
	}

}
