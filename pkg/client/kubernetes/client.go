package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	kitlog "github.com/go-kit/log"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func NewConfig(kubeconfig, context string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if len(context) > 0 {
		overrides.CurrentContext = context
	}

	if len(kubeconfig) > 0 {
		rules.ExplicitPath = kubeconfig
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}

func NewClient(kubeconfig, context string, logger kitlog.Logger) (kubernetes.Interface, error) {
	config, err := NewConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	logger.Log(
		"msg", "created new kubernetes client",
		"host", config.Host,
		"v", 3,
	)

	return clientset, nil
}

func NewClientConfigV1(name, user, url string, key, cert, ca []byte, token string) clientcmdapiv1.Config {
	return clientcmdapiv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: name,
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: name,
				Cluster: clientcmdapiv1.Cluster{
					Server:                   url,
					CertificateAuthorityData: ca,
				},
			},
		},
		Contexts: []clientcmdapiv1.NamedContext{
			{
				Name: name,
				Context: clientcmdapiv1.Context{
					Cluster:  name,
					AuthInfo: user,
				},
			},
		},
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			{
				Name: user,
				AuthInfo: clientcmdapiv1.AuthInfo{
					ClientCertificateData: cert,
					ClientKeyData:         key,
					Token:                 token,
				},
			},
		},
	}
}

func NewClientConfigV1OIDC(name, user, url, clientID, secret, issuer string, ca []byte) clientcmdapiv1.Config {
	return clientcmdapiv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: name,
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: name,
				Cluster: clientcmdapiv1.Cluster{
					Server:                   url,
					CertificateAuthorityData: ca,
				},
			},
		},
		Contexts: []clientcmdapiv1.NamedContext{
			{
				Name: name,
				Context: clientcmdapiv1.Context{
					Cluster:  name,
					AuthInfo: user,
				},
			},
		},
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			{
				Name: user,

				AuthInfo: clientcmdapiv1.AuthInfo{
					AuthProvider: &clientcmdapiv1.AuthProviderConfig{
						Name: "oidc",
						Config: map[string]string{
							"client-id":      clientID,
							"client-secret":  secret,
							"idp-issuer-url": issuer,
						},
					},
				},
			},
		},
	}
}

func EnsureCRD(clientset apiextensionsclient.Interface, logger kitlog.Logger) error {
	supportsV1 := false
	klusterCRDName := kubernikus_v1.KlusterResourcePlural + "." + kubernikus_v1.GroupName

	resourceList, err := clientset.Discovery().ServerResourcesForGroupVersion("apiextensions.k8s.io/v1")
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		for _, res := range resourceList.APIResources {
			if res.Kind == "CustomResourceDefinition" {
				supportsV1 = true
			}
		}
	}

	if supportsV1 {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: klusterCRDName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: kubernikus_v1.GroupName,
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    kubernikus_v1.SchemeGroupVersion.Version,
						Served:  true,
						Storage: true,
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"spec": {
										Type: "object",
									},
									"status": {
										Type: "object",
									},
								},
							},
						},
					},
				},
				Scope: apiextensionsv1.NamespaceScoped,
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: kubernikus_v1.KlusterResourcePlural,
					Kind:   reflect.TypeOf(kubernikus_v1.Kluster{}).Name(),
				},
			},
		}
		_, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), crd, metav1.CreateOptions{})
		//TODO: Should this error if it already exit?
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		// wait for CRD being established
		err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) { //nolint:staticcheck
			crd, err = clientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), klusterCRDName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			for _, cond := range crd.Status.Conditions {
				switch cond.Type {
				case apiextensionsv1.Established:
					if cond.Status == apiextensionsv1.ConditionTrue {
						return true, err
					}
				case apiextensionsv1.NamesAccepted:
					if cond.Status == apiextensionsv1.ConditionFalse {
						logger.Log(
							"msg", "name conflict while ensuring CRD",
							"reason", cond.Reason,
						)
					}
				}
			}
			return false, err
		})
		if err != nil {
			return fmt.Errorf("kluster CRD V1 couldn't be established: %v", err)
		}

	} else {

		crd := &apiextensionsv1beta1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: klusterCRDName,
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   kubernikus_v1.GroupName,
				Version: kubernikus_v1.SchemeGroupVersion.Version,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Plural: kubernikus_v1.KlusterResourcePlural,
					Kind:   reflect.TypeOf(kubernikus_v1.Kluster{}).Name(),
				},
			},
		}
		_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.TODO(), crd, metav1.CreateOptions{})
		//TODO: Should this error if it already exit?
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		// wait for CRD being established
		err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) { //nolint:staticcheck
			crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(context.TODO(), klusterCRDName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			for _, cond := range crd.Status.Conditions {
				switch cond.Type {
				case apiextensionsv1beta1.Established:
					if cond.Status == apiextensionsv1beta1.ConditionTrue {
						return true, err
					}
				case apiextensionsv1beta1.NamesAccepted:
					if cond.Status == apiextensionsv1beta1.ConditionFalse {
						logger.Log(
							"msg", "name conflict while ensuring CRD",
							"reason", cond.Reason,
						)
					}
				}
			}
			return false, err
		})
		if err != nil {
			return fmt.Errorf("kluster CRD V1beta1 couldn't be established: %v", err)
		}
	}

	return nil
}

func WaitForServer(client kubernetes.Interface, stopCh <-chan struct{}, logger kitlog.Logger) error {
	var healthzContent string

	err := wait.PollUntil(time.Second, func() (bool, error) { //nolint:staticcheck
		healthStatus := 0
		resp := client.Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.TODO()).StatusCode(&healthStatus)
		if healthStatus != http.StatusOK {
			logger.Log(
				"msg", "server isn't health yet. Waiting a little while.",
			)
			return false, nil
		}
		content, _ := resp.Raw()
		healthzContent = string(content)

		return true, nil
	}, stopCh)
	if err != nil {
		return fmt.Errorf("failed to contact apiserver. last health: %v error: %v", healthzContent, err)
	}

	return nil
}
