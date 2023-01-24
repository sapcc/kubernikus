package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	PVCTestName                        = "pvc-hostname"
	TestWaitForPVCBoundTimeout         = 5 * time.Minute
	TestWaitForPVCPodsRunning          = 15 * time.Minute
	TestWaitForPVCResizeInPlaceTimeOut = 5 * time.Minute
	TestWaitForSnapshotInPlaceTimeOut  = 5 * time.Minute
)

type VolumeTests struct {
	Kubernetes *framework.Kubernetes
	Namespace  string
	Nodes      *v1.NodeList
}

func (v *VolumeTests) Run(t *testing.T) {
	runParallel(t)

	v.Namespace = generator.SimpleNameGenerator.GenerateName("e2e-volumes-")

	var err error
	v.Nodes, err = v.Kubernetes.ClientSet.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{})
	require.NoError(t, err, "There must be no error while listing the kluster's nodes")
	require.NotEmpty(t, v.Nodes.Items, "No nodes returned by list")

	//defer t.Run("Cleanup", v.DeleteNamespace)
	t.Run("CreateNamespace", v.CreateNamespace)
	t.Run("WaitNamespace", v.WaitForNamespace)
	t.Run("CreatePVC", v.CreatePVC)
	t.Run("CreatePod", v.CreatePod)
	t.Run("PVCTests", func(t *testing.T) {
		t.Run("WaitPVCBound", v.WaitForPVCBound)
		t.Run("WaitPodRunning", v.WaitForPVCPodsRunning)
		// t.Run("WaitPVCResize", v.WaitForPVCResize)
		t.Run("WaitSnapshot", v.WaitForSnapshot)
	})

}

func (p *VolumeTests) CreateNamespace(t *testing.T) {
	_, err := p.Kubernetes.ClientSet.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: p.Namespace}}, meta_v1.CreateOptions{})
	require.NoError(t, err, "There must be no error while creating a namespace")
}

func (p *VolumeTests) WaitForNamespace(t *testing.T) {
	err := p.Kubernetes.WaitForDefaultServiceAccountInNamespace(p.Namespace)
	require.NoError(t, err, "There must be no error while waiting for the namespace")
}

func (p *VolumeTests) DeleteNamespace(t *testing.T) {
	err := p.Kubernetes.ClientSet.CoreV1().Namespaces().Delete(context.Background(), p.Namespace, meta_v1.DeleteOptions{})
	require.NoError(t, err, "There must be no error while deleting a namespace")
}

func (p *VolumeTests) CreatePod(t *testing.T) {
	_, err := p.Kubernetes.ClientSet.CoreV1().Pods(p.Namespace).Create(context.Background(), &v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      PVCTestName,
			Namespace: p.Namespace,
			Labels: map[string]string{
				"app": PVCTestName,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: ServeHostnameImage,
					Name:  PVCTestName,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      PVCTestName,
							MountPath: "/mymount",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: PVCTestName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: PVCTestName,
						},
					},
				},
			},
		},
	}, meta_v1.CreateOptions{})
	assert.NoError(t, err, "There should be no error while creating a pod with a volume")
}

func (p *VolumeTests) WaitForPVCPodsRunning(t *testing.T) {
	label := labels.SelectorFromSet(labels.Set(map[string]string{"app": PVCTestName}))
	_, err := p.Kubernetes.WaitForPodsWithLabelRunningReady(p.Namespace, label, 1, TestWaitForPVCPodsRunning)
	require.NoError(t, err, "There must be no error while waiting for the pod with mounted volume to become ready")
}

func (p *VolumeTests) CreatePVC(t *testing.T) {
	_, err := p.Kubernetes.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Create(context.Background(), &v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      PVCTestName,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}, meta_v1.CreateOptions{})
	assert.NoError(t, err)
}

func (p *VolumeTests) WaitForPVCBound(t *testing.T) {
	err := p.Kubernetes.WaitForPVCBound(p.Namespace, PVCTestName, TestWaitForPVCBoundTimeout)
	require.NoError(t, err, "There must be no error while waiting for the PVC to be bound")
}

func (p *VolumeTests) WaitForPVCResize(t *testing.T) {
	pvc, getErr := p.Kubernetes.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Get(context.Background(), PVCTestName, meta_v1.GetOptions{})
	require.NoError(t, getErr, "There must be no error getting the formerly created PVC")

	pvc.Spec.Resources.Requests["storage"] = resource.MustParse("2Gi")
	_, updateErr := p.Kubernetes.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Update(context.Background(), pvc, meta_v1.UpdateOptions{})
	require.NoError(t, updateErr, "There must be no error updating the PVC")

	waitForResizeErr := wait.PollImmediate(PollInterval, TestWaitForPVCResizeInPlaceTimeOut,
		func() (bool, error) {
			pvc, getErr := p.Kubernetes.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Get(context.Background(), PVCTestName, meta_v1.GetOptions{})
			if getErr != nil {
				return false, getErr
			}

			storageResized := *pvc.Status.Capacity.Storage() == resource.MustParse("2Gi") && pvc.Status.Phase == v1.PersistentVolumeClaimPhase("Bound")
			return storageResized, nil
		})
	require.NoError(t, waitForResizeErr, "There must be no error waiting for the PVC to be resized")
}

func (p *VolumeTests) WaitForSnapshot(t *testing.T) {
	const snapshotName = "volume-snapshot-e2e"

	dynamicClient, clientErr := dynamic.NewForConfig(p.Kubernetes.RestConfig)
	require.NoError(t, clientErr, "There must be no error creating the dynamic client")

	snapshotGvr := schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}

	snapShot := &unstructured.Unstructured{}
	snapShot.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "snapshot.storage.k8s.io/v1",
		"kind":       "VolumeSnapshot",
		"metadata": map[string]interface{}{
			"name": snapshotName,
		},
		"spec": map[string]interface{}{
			"volumeSnapshotClassName": "csi-cinder-snapclass",
			"source": map[string]interface{}{
				"persistentVolumeClaimName": PVCTestName,
			},
		},
	})

	_, createSnapshotErr := dynamicClient.Resource(snapshotGvr).Namespace(p.Namespace).Create(context.TODO(), snapShot, meta_v1.CreateOptions{})
	require.NoError(t, createSnapshotErr, "There must be no error creating the snapshot")

	deletePodErr := p.Kubernetes.ClientSet.CoreV1().Pods(p.Namespace).Delete(context.Background(), PVCTestName, meta_v1.DeleteOptions{})
	require.NoError(t, deletePodErr, "There must be no error deleting the pod")

	waitForSnapshotErr := wait.PollImmediate(PollInterval, TestWaitForSnapshotInPlaceTimeOut,
		func() (bool, error) {
			snapshot, getSnapshotErr := dynamicClient.Resource(snapshotGvr).Namespace(p.Namespace).Get(context.Background(), snapshotName, meta_v1.GetOptions{})
			if getSnapshotErr != nil {
				return false, getSnapshotErr
			}
			status, ok := snapshot.Object["status"].(map[string]interface{})
			if !ok {
				return false, nil
			}
			readyToUse, ok := status["readyToUse"].(bool)
			if !ok {
				return false, nil
			}

			return readyToUse, nil
		})
	require.NoError(t, waitForSnapshotErr, "The snapshot must be ready to use")

	deleteSnapShotErr := dynamicClient.Resource(snapshotGvr).Namespace(p.Namespace).Delete(context.Background(), snapshotName, meta_v1.DeleteOptions{})
	require.NoError(t, deleteSnapShotErr, "There must be no error deleting the snapshot")
}
