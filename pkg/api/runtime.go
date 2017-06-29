package api

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Runtime struct {
	Client    *kubernetes.Clientset
	TPRClient *rest.RESTClient
	TPRScheme *runtime.Scheme
}
