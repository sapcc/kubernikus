package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	TestWaitForPVCBoundTimeout = 10 * time.Minute
)

type VolumeTests struct {
	Kubernetes *framework.Kubernetes
	Nodes      *v1.NodeList
	Pods       *v1.PodList
	Namespace  string
}

func (p *VolumeTests) CreateNamespace(t *testing.T) {
	_, err := p.Kubernetes.ClientSet.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: p.Namespace}})
	require.NoError(t, err, "There must be no error while creating a namespace")
}

func (p *VolumeTests) WaitForNamespace(t *testing.T) {
	err := p.Kubernetes.WaitForDefaultServiceAccountInNamespace(p.Namespace)
	require.NoError(t, err, "There must be no error while waiting for the namespace")
}

func (p *VolumeTests) DeleteNamespace(t *testing.T) {
	err := p.Kubernetes.ClientSet.CoreV1().Namespaces().Delete(p.Namespace, nil)
	require.NoError(t, err, "There must be no error while deleting a namespace")
}

func (p *VolumeTests) CreatePod(t *testing.T) {
	nodeName := p.Nodes.Items[0].Name
	t.Run(nodeName, func(t *testing.T) {
		_, err := p.Kubernetes.ClientSet.CoreV1().Pods(p.Namespace).Create(&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", nodeName),
				Namespace:    p.Namespace,
				Labels: map[string]string{
					"app":  "pvc-hostname",
					"node": nodeName,
				},
			},
			Spec: v1.PodSpec{
				NodeName: nodeName,
				Containers: []v1.Container{
					{
						Image: ServeHostnameImage,
						Name:  "pvc-hostname",
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "pvc-hostname",
								MountPath: "/mymount",
							},
						},
					},
				},
				Volumes: []v1.Volume{
					{
						Name: "pvc-hostname",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc-hostname",
							},
						},
					},
				},
			},
		})
		assert.NoError(t, err, "There should be no error while creating a pod with a volume")
	})
}

func (p *VolumeTests) WaitForPodsRunning(t *testing.T) {
	label := labels.SelectorFromSet(labels.Set(map[string]string{"app": "pvc-hostname"}))
	_, err := p.Kubernetes.WaitForPodsWithLabelRunningReady(p.Namespace, label, 1, TestWaitForPodsRunningTimeout)
	require.NoError(t, err, "There must be no error while waiting for the pod with mounted volume to become ready")
}

func (p *VolumeTests) CreatePVC(t *testing.T) {
	_, err := p.Kubernetes.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Create(&v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      "pvc-hostname",
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
	})
	assert.NoError(t, err)
}

func (p *VolumeTests) WaitForPVCBound(t *testing.T) {
	err := p.Kubernetes.WaitForPVCBound(p.Namespace, "pvc-hostname", TestWaitForPVCBoundTimeout)
	require.NoError(t, err, "There must be no error while waiting for the PVC to be bound")
}
