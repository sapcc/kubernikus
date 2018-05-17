package api

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	clusterapi "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

type Runtime struct {
	Kubernikus clientset.Interface
	Kubernetes kubernetes.Interface
	ClusterAPI clusterapi.Interface
	Namespace  string
	Logger     log.Logger
}
