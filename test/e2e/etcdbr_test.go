package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	EtcdRestorePollInterval = 5 * time.Second
	EtcdRestoreTimeout      = 1 * time.Minute
	EtcdDataDir             = "/var/lib/etcd"
)

type EtcdBackupTests struct {
	KubernikusControlPlane *framework.Kubernikus
	KubernetesControlPlane *framework.Kubernetes
	KlusterName            string
	Namespace              string
}

func (e *EtcdBackupTests) Run(t *testing.T) {
	t.Parallel()
	t.Run("WaitForBackupRestore", e.WaitForBackupRestore)
}

func (e *EtcdBackupTests) WaitForBackupRestore(t *testing.T) {
	t.Parallel()

	UID, err := e.getServiceAccountUID("default")
	assert.NoError(t, err, "Error retrieving secret: %s", err)
	assert.NotEmpty(t, UID, "ServiceAccount UID is empty")

	opts := meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s-etcd", e.KlusterName),
	}
	pods, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).List(opts)
	assert.NoError(t, err, "Error retrieving etcd pod: %s", err)
	assert.NotEqual(t, 1, len(pods.Items), "There should only be one etcd pod")

	cmd := fmt.Sprintf("rm -rf %s/*", EtcdDataDir)
	_, _, err = e.KubernetesControlPlane.ExecShellInPodWithFullOutput(e.Namespace, pods.Items[0].Name, cmd)
	assert.NoError(t, err, "Deletion of etcd data failed: %s", err)

	var newUID string
	wait.PollImmediate(EtcdRestorePollInterval, EtcdRestoreTimeout,
		func() (bool, error) {
			newUID, _ = e.getServiceAccountUID("default")
			if UID == newUID {
				return true, nil
			}

			return false, nil
		})

	assert.EqualValues(t, UID, newUID, "Recovery of etcd backup failed: %s != %s", UID, newUID)
}

func (e *EtcdBackupTests) getServiceAccountUID(serviceAccountName string) (string, error) {
	serviceAccount, err := e.KubernetesControlPlane.ClientSet.CoreV1().ServiceAccounts(e.Namespace).Get(serviceAccountName, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(serviceAccount.GetUID()), nil
}
