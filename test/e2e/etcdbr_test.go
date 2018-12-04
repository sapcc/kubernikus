package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	EtcdFailPollInterval    = 1 * time.Second
	EtcdFailTimeout         = 90 * time.Second
	EtcdRestorePollInterval = 1 * time.Second
	EtcdRestoreTimeout      = 90 * time.Second
	EtcdDataDir             = "/var/lib/etcd/new.etcd"
)

type EtcdBackupTests struct {
	KubernetesControlPlane *framework.Kubernetes
	Kubernetes             *framework.Kubernetes
	FullKlusterName        string
	Namespace              string
}

func (e *EtcdBackupTests) Run(t *testing.T) {
	t.Run("WaitForBackupRestore", e.WaitForBackupRestore)
}

func (e *EtcdBackupTests) WaitForBackupRestore(t *testing.T) {
	err := e.Kubernetes.WaitForDefaultServiceAccountInNamespace("default")
	require.NoError(t, err, "There must be no error while waiting for the namespace")

	UID, err := e.getServiceAccountUID("default", "default")
	require.NoError(t, err, "Error retrieving default secret")
	require.NotEmpty(t, UID, "ServiceAccount UID should not be empty")

	opts := meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s-etcd", e.FullKlusterName),
	}
	pods, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).List(opts)
	require.NoError(t, err, "Error retrieving etcd pod: %s", err)
	require.EqualValues(t, 1, len(pods.Items), "There should be exactly one etcd pod, %d found", len(pods.Items))
	podName := pods.Items[0].GetName()
	require.NotEmpty(t, podName, "Podname should not be empty")

	pod, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(podName, meta_v1.GetOptions{})
	require.NoError(t, err, "Error retrieving resource version")
	rv := pod.GetResourceVersion()
	require.NotEmpty(t, rv, "ResourceVersion should not be empty")

	cmd := fmt.Sprintf("rm -rf %s/*", EtcdDataDir)
	_, _, err = e.KubernetesControlPlane.ExecCommandInContainerWithFullOutput(e.Namespace, podName, "backup", "/bin/sh", "-c", cmd)
	require.NoError(t, err, "Deletion of etcd data failed: %s", err)

	newRv := string(rv)
	wait.PollImmediate(EtcdFailPollInterval, EtcdFailTimeout,
		func() (bool, error) {
			pod, _ := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(podName, meta_v1.GetOptions{})
			newRv = pod.GetResourceVersion()
			return (newRv != rv), nil
		})
	require.NotEqual(t, rv, newRv, "Etcd is still up, can't test recovery")

	var newUID string
	wait.PollImmediate(EtcdRestorePollInterval, EtcdRestoreTimeout,
		func() (bool, error) {
			newUID, _ = e.getServiceAccountUID("default", "default")
			return (UID == newUID), nil
		})
	require.EqualValues(t, UID, newUID, "Recovery of etcd backup failed")
}

func (e *EtcdBackupTests) getServiceAccountUID(namespace, serviceAccountName string) (string, error) {
	serviceAccount, err := e.Kubernetes.ClientSet.CoreV1().ServiceAccounts(namespace).Get(serviceAccountName, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(serviceAccount.GetUID()), nil
}
