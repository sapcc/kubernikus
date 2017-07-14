package api

import "github.com/sapcc/kubernikus/pkg/kube"

type Runtime struct {
	Clients *kube.ClientCache
}
