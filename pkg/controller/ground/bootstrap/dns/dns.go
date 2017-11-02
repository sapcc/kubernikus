package dns

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

const (
	SERVICE_ACCOUNT    = "kube-dns"
	CONFIGMAP          = "kube-dns"
	DEFAULT_REPOSITORY = "gcr.io/google_containers"
	DEFAULT_VERSION    = "1.14.5"
	DEFAULT_DOMAIN     = "cluster.local"
	DEFAULT_CLUSTER_IP = "198.18.254.254"
)

type DeploymentOptions struct {
	Repository string
	Version    string
	Domain     string
}

type ServiceOptions struct {
	ClusterIP string
}

func SeedKubeDNS(client clientset.Interface, repository, version, domain, clusterIP string) error {
	if repository == "" {
		repository = DEFAULT_REPOSITORY
	}

	if version == "" {
		version = DEFAULT_VERSION
	}

	if domain == "" {
		domain = DEFAULT_DOMAIN
	}

	if clusterIP == "" {
		return errors.New("Cluster IP for kube-dns service missing.")
	}

	if err := createKubeDNSServiceAccount(client); err != nil {
		return err
	}

	if err := createKubeDNSConfigMap(client); err != nil {
		return err
	}

	if err := createKubeDNSDeployment(client, repository, version, domain); err != nil {
		return err
	}

	if err := createKubeDNSService(client, clusterIP); err != nil {
		return err
	}

	return nil
}

func createKubeDNSServiceAccount(client clientset.Interface) error {
	return CreateOrUpdateServiceAccount(client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SERVICE_ACCOUNT,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"kubernetes.io/cluster-service":   "true",
				"addonmanager.kubernetes.io/mode": "Reconcile",
			},
		},
	})
}

func createKubeDNSConfigMap(client clientset.Interface) error {
	return CreateOrUpdateConfigMap(client, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CONFIGMAP,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"addonmanager.kubernetes.io/mode": "EnsureExists",
			},
		},
	})
}

func createKubeDNSDeployment(client clientset.Interface, repository, version, domain string) error {
	options := &DeploymentOptions{
		Repository: repository,
		Version:    version,
		Domain:     domain,
	}

	deployment, err := getKubeDNSDeployment(options)
	if err != nil {
		return err
	}

	if err := CreateOrUpdateDeployment(client, deployment); err != nil {
		return err
	}

	return nil
}

func createKubeDNSService(client clientset.Interface, clusterIP string) error {
	options := &ServiceOptions{
		ClusterIP: clusterIP,
	}

	service, err := getKubeDNSService(options)
	if err != nil {
		return err
	}

	if err := CreateOrUpdateService(client, service); err != nil {
		return err
	}

	return nil
}

func getKubeDNSDeployment(options *DeploymentOptions) (*apps.Deployment, error) {
	manifest := KubeDNSDeployment_v20171016
	deployment := &apps.Deployment{}

	template, err := RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	if err := runtime.DecodeInto(api.Codecs.UniversalDecoder(), template, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

func getKubeDNSService(options *ServiceOptions) (*v1.Service, error) {
	manifest := KubeDNSService_v20171016
	service := &v1.Service{}

	template, err := RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	if err := runtime.DecodeInto(api.Codecs.UniversalDecoder(), template, service); err != nil {
		return nil, err
	}

	return service, nil
}

func CreateOrUpdateServiceAccount(client clientset.Interface, sa *v1.ServiceAccount) error {
	if _, err := client.CoreV1().ServiceAccounts(sa.ObjectMeta.Namespace).Create(sa); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create serviceaccount: %v", err)
		}
	}
	return nil
}

func CreateOrUpdateDeployment(client clientset.Interface, deploy *apps.Deployment) error {
	if _, err := client.AppsV1beta1().Deployments(deploy.ObjectMeta.Namespace).Create(deploy); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create deployment: %v", err)
		}

		if _, err := client.AppsV1beta1().Deployments(deploy.ObjectMeta.Namespace).Update(deploy); err != nil {
			return fmt.Errorf("unable to update deployment: %v", err)
		}
	}
	return nil
}

func CreateOrUpdateService(client clientset.Interface, service *v1.Service) error {
	if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Create(service); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create a new kube-dns service: %v", err)
		}

		if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Update(service); err != nil {
			return fmt.Errorf("unable to create/update the kube-dns service: %v", err)
		}
	}
	return nil
}

func CreateOrUpdateConfigMap(client clientset.Interface, configmap *v1.ConfigMap) error {
	if _, err := client.CoreV1().ConfigMaps(configmap.ObjectMeta.Namespace).Create(configmap); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configmap: %v", err)
		}

		if _, err := client.CoreV1().ConfigMaps(configmap.ObjectMeta.Namespace).Update(configmap); err != nil {
			return fmt.Errorf("unable to update configmap: %v", err)
		}
	}
	return nil
}

func RenderManifest(strtmpl string, obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(strtmpl)
	if err != nil {
		return nil, fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return nil, fmt.Errorf("error when executing template: %v", err)
	}
	return buf.Bytes(), nil
}
