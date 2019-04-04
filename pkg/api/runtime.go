package api

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	"github.com/sapcc/kubernikus/pkg/version"
)

type Runtime struct {
	Kubernikus clientset.Interface
	Kubernetes kubernetes.Interface
	Namespace  string
	Logger     log.Logger
	Images     *version.ImageRegistry
}
