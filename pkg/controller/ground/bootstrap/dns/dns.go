package dns

import (
	"errors"

	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
)

const (
	SERVICE_ACCOUNT    = "kube-dns"
	CONFIGMAP          = "kube-dns"
	DEFAULT_REPOSITORY = "sapcc" // Used to be gcr.io/google_containers but that is not working in china

	// If you change this version you need to republish the images:
	//   * k8s-dns-kube-dns-amd64
	//   * k8s-dns-sidecar-amd64
	//   * k8s-dns-dnsmasq-nanny-amd64
	//
	// Workflow:
	//   docker pull gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.9
	//   docker tag gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.9 sapcc/k8s-dns-kube-dns-amd64:1.14.9
	//   docker push sapcc/k8s-dns-kube-dns-amd64:1.14.9
	//
	DEFAULT_VERSION = "1.14.9"
)

var (
	DEFAULT_DOMAIN = spec.MustDefaultString("KlusterSpec", "dnsDomain")
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
	return bootstrap.CreateOrUpdateServiceAccount(client, &v1.ServiceAccount{
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
	return bootstrap.CreateOrUpdateConfigMap(client, &v1.ConfigMap{
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

	if err := bootstrap.CreateOrUpdateDeployment(client, deployment); err != nil {
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

	if err := bootstrap.CreateOrUpdateService(client, service); err != nil {
		return err
	}

	return nil
}

func getKubeDNSDeployment(options *DeploymentOptions) (*extensions.Deployment, error) {
	manifest := KubeDNSDeployment_v20171016

	template, err := bootstrap.RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	deployment, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &extensions.Deployment{})
	if err != nil {
		return nil, err
	}

	return deployment.(*extensions.Deployment), nil
}

func getKubeDNSService(options *ServiceOptions) (*v1.Service, error) {
	manifest := KubeDNSService_v20171016

	template, err := bootstrap.RenderManifest(manifest, options)
	if err != nil {
		return nil, err
	}

	service, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &v1.Service{})
	if err != nil {
		return nil, err
	}

	return service.(*v1.Service), nil
}
