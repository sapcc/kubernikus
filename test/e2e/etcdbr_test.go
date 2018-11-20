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
	EtcdFailPollInterval    = 1 * time.Second
	EtcdFailTimeout         = 60 * time.Second
	EtcdRestorePollInterval = 2 * time.Second
	EtcdRestoreTimeout      = 60 * time.Second
	EtcdDataDir             = "/var/lib/etcd/new.etcd"
)

type EtcdBackupTests struct {
	KubernikusControlPlane *framework.Kubernikus
	KubernetesControlPlane *framework.Kubernetes
	FullKlusterName        string
	Namespace              string
}

func (e *EtcdBackupTests) Run(t *testing.T) {
	t.Run("WaitForBackupRestore", e.WaitForBackupRestore)
}

func (e *EtcdBackupTests) WaitForBackupRestore(t *testing.T) {
	UID, err := e.getServiceAccountUID("default")
	assert.NoError(t, err, "Error retrieving secret: %s", err)
	assert.NotEmpty(t, UID, "ServiceAccount UID is empty")

	opts := meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s-etcd", e.FullKlusterName),
	}
	pods, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).List(opts)
	assert.NoError(t, err, "Error retrieving etcd pod: %s", err)
	assert.EqualValues(t, 1, len(pods.Items), "There should be exactly one etcd pod, %d found", len(pods.Items))
	podName := pods.Items[0].GetName()

	pod, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(podName, meta_v1.GetOptions{})
	assert.NoError(t, err, "Error retrieving resource version")
	rv := pod.GetResourceVersion()

	cmd := fmt.Sprintf("rm -rf %s/*", EtcdDataDir)
	_, _, err = e.KubernetesControlPlane.ExecCommandInContainerWithFullOutput(e.Namespace, podName, "backup", "/bin/sh", "-c", cmd)
	assert.NoError(t, err, "Deletion of etcd data failed: %s", err)

	newRv := string(rv)
	wait.PollImmediate(EtcdFailPollInterval, EtcdFailTimeout,
		func() (bool, error) {
			pod, _ := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(podName, meta_v1.GetOptions{})
			newRv = pod.GetResourceVersion()
			return (newRv != rv), nil
		})
	assert.NotEqual(t, rv, newRv, "Etcd is still up, can't test recovery")

	var newUID string
	wait.PollImmediate(EtcdRestorePollInterval, EtcdRestoreTimeout,
		func() (bool, error) {
			newUID, _ = e.getServiceAccountUID("default")
			return (UID == newUID), nil
		})

	assert.EqualValues(t, UID, newUID, "Recovery of etcd backup failed")
}

func (e *EtcdBackupTests) getServiceAccountUID(serviceAccountName string) (string, error) {
	serviceAccount, err := e.KubernetesControlPlane.ClientSet.CoreV1().ServiceAccounts(e.Namespace).Get(serviceAccountName, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(serviceAccount.GetUID()), nil
}
