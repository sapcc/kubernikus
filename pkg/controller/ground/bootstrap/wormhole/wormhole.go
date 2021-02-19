package wormhole

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

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
	if err := createDaemonSet(client, WormholeDaemonset, vars); err != nil {
		return err
	}

	return nil
}

func createDaemonSet(client clientset.Interface, manifest string, vars data) error {
	template, err := bootstrap.RenderManifest(manifest, vars)
	if err != nil {
		return err
	}

	daemonset, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &apps.DaemonSet{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateDaemonset(client, daemonset.(*apps.DaemonSet)); err != nil {
		return err
	}

	return nil
}
