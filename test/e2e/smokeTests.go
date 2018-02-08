package main

import (
	"fmt"
	"log"
	"os"

	"k8s.io/api/core/v1"
)

func (s *E2ETestSuite) SetupSmokeTest() {
	// double check if cluster is ready for smoke test or exit
	s.isClusterUpOrWait()
	s.createClientset()
	s.getReadyPods()
	s.getReadyNodes()
	s.isClusterBigEnoughForSmokeTest()
	s.cleanUp()
	s.createPods()
	s.createServices()
}

func (s *E2ETestSuite) RunSmokeTest() {
	if s.IsTestNetwork || s.IsTestSmoke || s.IsTestAll {
		s.TestPod2PodCommunication()
	}

	if s.IsTestVolume || s.IsTestSmoke || s.IsTestAll {
		s.TestAttachVolume()
	}

	log.Print("[passed smoke tests]")
}

func (s *E2ETestSuite) TestAttachVolume() {
	log.Printf("testing persistent volume attachment")
	s.createPVCForPod()
	s.createPodWithMount()
	s.writeFileToMountedVolume()
	s.readFileFromMountedVolume()
}

func (s *E2ETestSuite) TestPod2PodCommunication() {
	log.Print("testing network")

	log.Print("step 1: testing pod to pod")
	for _, source := range s.readyPods {
		for _, target := range s.readyPods {
			select {
			default:
				s.dialPodIP(&source, &target)
			case <-s.stopCh:
				return
			}
		}
	}

	log.Printf("step 2: testing pod to service IP")
	for _, source := range s.readyPods {
		for _, target := range s.readyServices {
			select {
			default:
				s.dialServiceIP(&source, &target)
			case <-s.stopCh:
				os.Exit(1)
			}
		}
	}

	log.Printf("step 3: testing pod to service name")
	for _, source := range s.readyPods {
		for _, target := range s.readyServices {
			select {
			default:
				s.dialServiceName(&source, &target)
			case <-s.stopCh:
				os.Exit(1)
			}
		}
	}

	log.Print("[network test done]")
}

func (s *E2ETestSuite) dialPodIP(source *v1.Pod, target *v1.Pod) {
	_, err := s.dial(source, target.Status.PodIP, NginxPort)
	result := "success"
	if err != nil {
		result = "failure"
	}

	resultMsg := fmt.Sprintf("[%v] node/%-15v --> node/%-15v   pod/%-15v --> pod/%-15v\n",
		result,
		source.Spec.NodeName,
		target.Spec.NodeName,
		source.Status.PodIP,
		target.Status.PodIP,
	)

	if result == "failure" {
		s.handleError(fmt.Errorf("%v \n error: %#v", resultMsg, err))
	} else {
		fmt.Printf(resultMsg)
	}
}

func (s *E2ETestSuite) dialServiceIP(source *v1.Pod, target *v1.Service) {
	_, err := s.dial(source, target.Spec.ClusterIP, NginxPort)
	result := "success"
	if err != nil {
		result = "failure"
	}
	resultMsg := fmt.Sprintf("[%v] node/%-15v --> node/%-15v   pod/%-15v --> svc/%-15v\n",
		result,
		source.Spec.NodeName,
		target.Labels["nodeName"],
		source.Status.PodIP,
		target.Spec.ClusterIP,
	)

	if result == "failure" {
		s.handleError(fmt.Errorf("%v \n error: %#v", resultMsg, err))
	} else {
		fmt.Printf(resultMsg)
	}
}

func (s *E2ETestSuite) dialServiceName(source *v1.Pod, target *v1.Service) {
	_, err := s.dial(source, fmt.Sprintf("%s.%s.svc", target.GetName(), target.GetNamespace()), NginxPort)
	result := "success"
	if err != nil {
		result = "failure"
	}
	resultMsg := fmt.Sprintf("[%v] node/%-15v --> node/%-15v   pod/%-15v --> svc/%-15v\n",
		result,
		source.Spec.NodeName,
		target.Labels["nodeName"],
		source.Status.PodIP,
		target.Spec.ClusterIP,
	)

	if result == "failure" {
		s.handleError(fmt.Errorf("%v \n error: %#v", resultMsg, err))
	} else {
		fmt.Printf(resultMsg)
	}
}

func (s *E2ETestSuite) dial(sourcePod *v1.Pod, targetIP string, targetPort int32) (string, error) {
	cmd := fmt.Sprintf("wget --timeout=%v -O - http://%v:%v", TimeoutWGET, targetIP, targetPort)
	return RunHostCmd(s.KubeConfig, sourcePod.GetNamespace(), sourcePod.GetName(), cmd)
}

func (s *E2ETestSuite) writeFileToMountedVolume() {
	cmd := fmt.Sprintf("echo hase > %v/myfile", PVCMountPath)
	_, err := RunHostCmd(s.KubeConfig, Namespace, PVCName, cmd)
	result := "success"
	if err != nil {
		result = "failure"
	}
	resultMsg := fmt.Sprintf("[%v] writing file %v/myfile", result, PVCMountPath)

	if result == "failure" {
		s.handleError(fmt.Errorf("%v \n error: %#v", resultMsg, err))
	} else if result != "" || result != `stderr: ""` {
		log.Println(resultMsg)
	}
}

func (s *E2ETestSuite) readFileFromMountedVolume() {
	cmd := fmt.Sprintf("cat %v/myfile", PVCMountPath)
	_, err := RunHostCmd(s.KubeConfig, Namespace, PVCName, cmd)
	result := "success"
	if err != nil {
		result = "failure"
	}
	resultMsg := fmt.Sprintf("[%v] reading file %v/myfile", result, PVCMountPath)

	if result == "failure" {
		s.handleError(fmt.Errorf("%v \n error: %#v", resultMsg, err))
	} else if result != "" || result != `stderr: ""` {
		log.Println(resultMsg)
	}
}
