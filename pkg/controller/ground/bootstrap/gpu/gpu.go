package gpu

import (
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

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

	daemonset, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &extensions.DaemonSet{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateDaemonset(client, daemonset.(*extensions.DaemonSet)); err != nil {
		return err
	}
	return nil
}
