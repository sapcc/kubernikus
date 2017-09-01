package api

import (
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	"k8s.io/client-go/kubernetes"
)

type Runtime struct {
	Clients *Clients
}

type Clients struct {
	Kubernikus clientset.Interface
	Kubernetes kubernetes.Interface
}
