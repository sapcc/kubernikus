package main

import (
	"fmt"
	"log"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"

	"k8s.io/client-go/tools/clientcmd"
)

func (s *E2ETestSuite) createClientset() {
	s.getClusterKubeConfig()

	config, err := clientcmd.Load([]byte(s.KubeConfig))
	s.handleError(err)

	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: config.CurrentContext,
	}

	kubeConfig := clientcmd.NewNonInteractiveClientConfig(
		*config,
		config.CurrentContext,
		configOverrides,
		nil,
	)

	clientConfig, err := kubeConfig.ClientConfig()
	s.handleError(err)

	clientSet, err := kubernetes.NewForConfig(clientConfig)
	s.handleError(err)

	s.clientSet = clientSet
}

func (s *E2ETestSuite) isClusterBigEnoughForSmokeTest() {
	nodeCount := len(s.readyNodes)
	if nodeCount < 2 {
		s.handleError(fmt.Errorf("[failure] found %v nodes in cluster. the smoke test requires a minimum of 2 nodes. aborting", nodeCount))
	}
}

func (s *E2ETestSuite) createPods() {
	for _, node := range s.readyNodes {
		//create pod
		pod, err := s.clientSet.CoreV1().Pods(Namespace).Create(&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", NginxName, node.Name),
				Namespace: Namespace,
				Labels: map[string]string{
					"app":      NginxName,
					"nodeName": node.Name,
					"test":     "e2e",
				},
			},
			Spec: v1.PodSpec{
				NodeName: node.Name,
				Containers: []v1.Container{
					{
						Image: NginxImage,
						Name:  NginxName,
						Ports: []v1.ContainerPort{
							{
								Name:          "http",
								ContainerPort: NginxPort,
							},
						},
					},
				},
			},
		})
		s.handleError(err)
		log.Printf("created pod %v/%v on node %v", pod.GetNamespace(), pod.GetName(), node.Name)

		// wait until ready
		w, err := s.clientSet.CoreV1().Pods(Namespace).Watch(meta_v1.SingleObject(
			meta_v1.ObjectMeta{
				Name: pod.GetName(),
			},
		))
		s.handleError(err)
		_, err = watch.Until(TimeoutPod, w, isPodRunning)
		s.handleError(err)
	}
	s.getReadyPods()
}

func (s *E2ETestSuite) createServices() {
	for _, node := range s.readyNodes {
		service := &v1.Service{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", NginxName, node.Name),
				Namespace: Namespace,
				Labels: map[string]string{
					"app":      NginxName,
					"podName":  NginxName,
					"nodeName": node.Name,
					"test":     "e2e",
				},
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Port:       NginxPort,
						TargetPort: intstr.FromInt(NginxPort),
					},
				},
				Type: v1.ServiceType(v1.ServiceTypeClusterIP),
				Selector: map[string]string{
					"app":      NginxName,
					"nodeName": node.Name,
				},
			},
		}

		log.Printf("created service %v/%v for pod on node %v", service.GetNamespace(), service.GetName(), node.Name)

		_, err := s.clientSet.CoreV1().Services(Namespace).Create(service)
		s.handleError(err)
	}
	s.getReadyServices()
}

func (s *E2ETestSuite) getReadyNodes() {
	nodes, err := s.clientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
	s.handleError(err)
	for _, node := range nodes.Items {
		log.Printf("found node: %s", node.Name)
	}
	s.readyNodes = nodes.Items
}

func (s *E2ETestSuite) getReadyPods() {
	pods, err := s.clientSet.CoreV1().Pods(Namespace).List(meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", NginxName),
	})
	s.handleError(err)
	for _, pod := range pods.Items {
		log.Printf("found pod %s/%s on node %s", pod.GetNamespace(), pod.GetName(), pod.Spec.NodeName)
	}
	s.readyPods = pods.Items
}

func (s *E2ETestSuite) getReadyServices() {
	services, err := s.clientSet.CoreV1().Services(Namespace).List(meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", NginxName),
	})
	s.handleError(err)
	for _, svc := range services.Items {
		log.Printf("found service %s/%s", svc.GetNamespace(), svc.GetName())
	}
	s.readyServices = services.Items
}

func (s *E2ETestSuite) createPVCForPod() {
	pvc, err := s.clientSet.CoreV1().PersistentVolumeClaims(Namespace).Create(&v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: Namespace,
			Name:      PVCName,
			Labels: map[string]string{
				"app":  PVCName,
				"test": "e2e",
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(PVCSize),
				},
			},
		},
	})
	s.handleError(err)
	log.Printf("created PVC %v/%v", pvc.GetNamespace(), pvc.GetName())
	_, err = s.waitForPVC(pvc)
	s.handleError(err)

	log.Printf("waiting for PVC %v/%v to be available", pvc.GetNamespace(), pvc.GetName())
}

func (s *E2ETestSuite) createPodWithMount() {
	nodeName := s.readyNodes[0].GetName()
	pod, err := s.clientSet.CoreV1().Pods(Namespace).Create(&v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      PVCName,
			Namespace: Namespace,
			Labels: map[string]string{
				"app":      PVCName,
				"nodeName": nodeName,
				"test":     "e2e",
			},
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
			Containers: []v1.Container{
				{
					Image: NginxImage,
					Name:  PVCName,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      PVCName,
							MountPath: PVCMountPath,
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: PVCName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: PVCName,
						},
					},
				},
			},
		},
	})
	s.handleError(err)
	log.Printf("created pod %v/%v with mounted volume", pod.GetNamespace(), pod.GetName())

	log.Printf("waiting for pod %v/%v to become ready", pod.GetNamespace(), pod.GetName())
	// wait until ready
	w, err := s.clientSet.CoreV1().Pods(Namespace).Watch(meta_v1.SingleObject(
		meta_v1.ObjectMeta{
			Name: pod.GetName(),
		},
	))
	s.handleError(err)
	_, err = watch.Until(TimeoutPod, w, isPodRunning)
	s.handleError(err)
}

func (s *E2ETestSuite) cleanUp() {
	log.Printf("cleaning up before running smoke tests")
	pods, err := s.clientSet.CoreV1().Pods(Namespace).List(meta_v1.ListOptions{
		LabelSelector: "test=e2e",
	})
	if err != nil {
		log.Fatalf("error while cleaning up smoke tests pods %v", err)
	}
	// pods
	for _, pod := range pods.Items {
		if err = s.clientSet.CoreV1().Pods(pod.GetNamespace()).Delete(pod.GetName(), &meta_v1.DeleteOptions{}); err != nil {
			log.Printf("could not delete pod %v/%v", pod.GetNamespace(), pod.GetName())
		}
		_, err = s.waitForPodDeleted(pod.GetNamespace(), pod.GetName())
		if err != nil {
			log.Print(err)
		}
	}
	// services
	svcs, err := s.clientSet.CoreV1().Services(Namespace).List(meta_v1.ListOptions{
		LabelSelector: "test=e2e",
	})
	if err != nil {
		log.Printf("error while cleaning smoke tests services %v", err)
	}
	for _, svc := range svcs.Items {
		if err = s.clientSet.CoreV1().Services(svc.GetNamespace()).Delete(svc.GetName(), &meta_v1.DeleteOptions{}); err != nil {
			log.Printf("could not delete service %v/%v", svc.GetNamespace(), svc.GetName())
		}
	}
	// pvcs
	pvcs, err := s.clientSet.CoreV1().PersistentVolumeClaims(Namespace).List(meta_v1.ListOptions{
		LabelSelector: "test=e2e",
	})
	if err != nil {
		log.Printf("error while cleaning smoke tests pvc %v", err)
	}
	for _, pvc := range pvcs.Items {
		if err = s.clientSet.CoreV1().PersistentVolumeClaims(pvc.GetNamespace()).Delete(pvc.GetName(), &meta_v1.DeleteOptions{}); err != nil {
			log.Printf("could not delete pvc %v/%v", pvc.GetNamespace(), pvc.GetName())
		}
	}
}

func (s *E2ETestSuite) getClusterKubeConfig() {
	credentials, err := s.kubernikusClient.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(ClusterName),
		s.authFunc(),
	)
	s.handleError(err)
	cfg := credentials.Payload.Kubeconfig
	if cfg == "" {
		s.handleError(fmt.Errorf("kubeconfig of cluster %s is empty", ClusterName))
	}
	s.KubeConfig = cfg
}
