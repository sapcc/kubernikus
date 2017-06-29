package kube

import (
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
)

func NewTPRClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := tprv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &tprv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, scheme, nil
}

func EnsureTPR(clientset kubernetes.Interface) error {
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kluster." + tprv1.GroupName,
		},
		Versions: []v1beta1.APIVersion{
			{Name: tprv1.SchemeGroupVersion.Version},
		},
		Description: "Managed kubernetes cluster",
	}

	_, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func WaitForTPR(client *rest.RESTClient) error {
	return wait.Poll(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		_, err := client.Get().Namespace(apiv1.NamespaceDefault).Resource(tprv1.KlusterResourcePlural).DoRaw()
		if err == nil {
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
}
