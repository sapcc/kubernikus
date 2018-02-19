package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
)

func (s *E2ETestSuite) SetupSmokeTest() {
	// double check if cluster is ready for smoke test or exit
	s.isClusterUpOrWait()
	s.TestSetupKubernikusCtl()
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

	log.Print("[passed] smoke tests")
}

func (s *E2ETestSuite) TestSetupKubernikusCtl() {
	log.Printf("Setting up kubernikusctl")
	// get kluster info which contains the URL to the kubernikusctl binary and download it
	if runtime.GOOS == "darwin" {
		log.Println("Detected dev machine. skipping kubernikusctl download. make sure 'kubernikusctl' is installed.")
	} else {
		if err := s.getKubernikusctlBinary(); err != nil {
			s.handleError(fmt.Errorf("[failure] could not get kubernikusctl. reason: %v", err))
		}
	}
	// auth init to get kubeconfig for kluster
	if err := s.initKubernikusctl(); err != nil {
		s.handleError(fmt.Errorf("[failure] could not 'kubernikusctl auth init'. reason: %v", err))
	}
}

func (s *E2ETestSuite) getKubernikusctlBinary() error {
	log.Printf("getting info for kluster %s to obtain link to kubernikusctl", s.ClusterName)
	info, err := s.kubernikusClient.Operations.GetClusterInfo(operations.NewGetClusterInfoParams().WithName(s.ClusterName), s.authFunc())
	if err != nil {
		return err
	}
	for _, b := range info.Payload.Binaries {
		if b.Name == "kubernikusctl" {
			for _, l := range b.Links {
				if l.Platform == runtime.GOOS {
					filePath := fmt.Sprintf("%s/%s", PathBin, KubernikusctlBinaryName)

					p := PathBin + ":" + os.Getenv("PATH")
					if err := os.Setenv("PATH", p); err != nil {
						return err
					}
					log.Printf("updated PATH=%v", os.Getenv("PATH"))

					log.Printf("Downloading %s", l.Link)
					resp, err := http.Get(l.Link)
					if err != nil {
						return err
					}
					defer resp.Body.Close()

					fOut, err := os.Create(filePath)
					if err != nil {
						return err
					}
					defer fOut.Close()

					_, err = io.Copy(fOut, resp.Body)
					if err != nil {
						return err
					}

					if err := fOut.Chmod(0777); err != nil {
						return err
					}
					// make sure the file is closed before using it
					fOut.Close()

					path, err := exec.LookPath(KubernikusctlBinaryName)
					if err != nil {
						return err
					}
					log.Printf("found %s", path)

					_, err = RunKubernikusctlHostCmd("--help")
					return err
				}
			}
		}
	}
	return fmt.Errorf("no link for kubernikusctl binary found for os: %v", runtime.GOOS)
}

func (s *E2ETestSuite) initKubernikusctl() error {
	// gets auth params from env
	out, err := RunKubernikusctlHostCmd("auth", "init")
	log.Println(out)
	return err
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
	cmd := fmt.Sprintf("wget --tries=%v --timeout=%v --retry-connrefused -O - http://%v:%v", WGETRetries, WGETTimeout, targetIP, targetPort)
	return RunKubectlHostCmd(sourcePod.GetNamespace(), sourcePod.GetName(), cmd)
}

func (s *E2ETestSuite) writeFileToMountedVolume() {
	cmd := fmt.Sprintf("echo hase > %v/myfile", PVCMountPath)
	_, err := RunKubectlHostCmd(Namespace, PVCName, cmd)
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
	_, err := RunKubectlHostCmd(Namespace, PVCName, cmd)
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
