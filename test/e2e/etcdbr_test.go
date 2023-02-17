package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	podutil "github.com/sapcc/kubernikus/pkg/util/pod"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	EtcdFailPollInterval    = 1 * time.Second
	EtcdFailTimeout         = 90 * time.Second
	EtcdRestorePollInterval = 1 * time.Second
	EtcdRestoreTimeout      = 3 * time.Minute
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

	etcdPod, err := e.GetPod(fmt.Sprintf("app=%s-etcd", e.FullKlusterName))
	require.NoError(t, err, "Error retrieving etcd pod: %s", err)

	rv := etcdPod.GetResourceVersion()
	require.NotEmpty(t, rv, "ResourceVersion should not be empty")

	apiPod, err := e.GetPod(fmt.Sprintf("app=%s-apiserver", e.FullKlusterName))
	require.NoError(t, err, "Error retrieving apiserver pod: %s", err)

	cmd := fmt.Sprintf("mv %s %s.bak", EtcdDataDir, EtcdDataDir)
	_, _, err = e.KubernetesControlPlane.ExecCommandInContainerWithFullOutput(e.Namespace, etcdPod.Name, "backup", "/bin/sh", "-c", cmd)
	require.NoError(t, err, "Deletion of etcd data failed: %s", err)

	newRv := string(rv)
	wait.PollImmediate(EtcdFailPollInterval, EtcdFailTimeout,
		func() (bool, error) {
			pod, _ := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(context.Background(), etcdPod.Name, meta_v1.GetOptions{})
			newRv = pod.GetResourceVersion()
			return (newRv != rv), nil
		})
	require.NotEqual(t, rv, newRv, "Etcd is still up, can't test recovery")

	var newUID string
	err = wait.PollImmediate(EtcdRestorePollInterval, EtcdRestoreTimeout,
		func() (bool, error) {
			newUID, _ = e.getServiceAccountUID("default", "default")
			return (UID == newUID), nil
		})
	require.NoError(t, err)
	require.EqualValues(t, UID, newUID, "Recovery of etcd backup failed")

	err = wait.PollImmediate(EtcdRestorePollInterval, EtcdRestoreTimeout,
		func() (bool, error) {
			p, _ := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).Get(context.Background(), apiPod.Name, meta_v1.GetOptions{})

			return p.Status.ContainerStatuses[0].RestartCount > apiPod.Status.ContainerStatuses[0].RestartCount && podutil.IsPodReady(p), nil
		})
	require.NoError(t, err, "apiserver did not restart after etcd restore")
}

func (e *EtcdBackupTests) getServiceAccountUID(namespace, serviceAccountName string) (string, error) {
	serviceAccount, err := e.Kubernetes.ClientSet.CoreV1().ServiceAccounts(namespace).Get(context.Background(), serviceAccountName, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(serviceAccount.GetUID()), nil
}

func (e *EtcdBackupTests) GetPod(labelSelector string) (*v1.Pod, error) {
	opts := meta_v1.ListOptions{
		LabelSelector: labelSelector,
	}

	pods, err := e.KubernetesControlPlane.ClientSet.CoreV1().Pods(e.Namespace).List(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("Failed to list pods: %w", err)
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("Expected to find one pod for selector %s, found %d", labelSelector, len(pods.Items))
	}
	return &pods.Items[0], nil
}
