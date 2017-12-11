package api

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/generated/clientset"
)

type Runtime struct {
	Kubernikus clientset.Interface
	Kubernetes kubernetes.Interface
	Namespace  string
	Logger     log.Logger
}
