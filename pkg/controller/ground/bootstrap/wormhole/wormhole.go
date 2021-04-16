package wormhole

import (
	"fmt"

	clientset "k8s.io/client-go/kubernetes"

	kubernikus "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type data struct {
	Image         string
	ApiserverIP   string
	ApiserverPort string
}

func SeedWormhole(client clientset.Interface, versions version.KlusterVersion, kluster *kubernikus.Kluster) error {
	vars := data{
		versions.Wormhole.Repository + ":" + version.GitCommit,
		kluster.Spec.AdvertiseAddress,
		fmt.Sprintf("%d", kluster.Spec.AdvertisePort),
	}
	if err := bootstrap.CreateDaemonSetFromTemplate(client, WormholeDaemonset, vars); err != nil {
		return err
	}

	return nil
}
