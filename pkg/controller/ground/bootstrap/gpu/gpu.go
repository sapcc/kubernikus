package gpu

import (
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	apps "k8s.io/api/apps/v1"

	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
)

func SeedGPUSupport(client clientset.Interface) error {
	if err := createDaemonSet(client, NVIDIADevicePlugin_v20180808); err != nil {
		return err
	}
	if err := createDaemonSet(client, NVIDIADriverInstaller_v20180808); err != nil {
		return err
	}
	return nil
}

func createDaemonSet(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
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
