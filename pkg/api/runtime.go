package api

import (
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
)

type Runtime struct {
	Clients *Clients
}

type Clients struct {
	Kubernikus clientset.Interface
}
