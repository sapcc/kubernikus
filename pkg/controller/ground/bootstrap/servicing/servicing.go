package servicing

import (
	clientset "k8s.io/client-go/kubernetes"

	kubernikus "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

func SeedDisableNodeServices(client clientset.Interface, versions version.KlusterVersion, kluster *kubernikus.Kluster) error {
	if err := bootstrap.CreateDaemonSetFromTemplate(client, DisableNodeServicesDaemonset, nil); err != nil {
		return err
	}

	return nil
}
