package handlers

import (
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/bootstraptoken"
)

var kubeletConfigurationTemplate = template.Must(template.New("config").Parse(`kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
readOnlyPort: 0
clusterDomain: {{ .ClusterDomain }}
clusterDNS: [{{ .ClusterDNS }}]
authentication:
  x509:
    clientCAFile: {{ .KubeletClientsCAFile }}
  anonymous:
    enabled: true
rotateCertificates: true
featureGates:
  NodeLease: false
`))

var kubeletConfigurationTemplate117 = template.Must(template.New("config").Parse(`kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
readOnlyPort: 0
clusterDomain: {{ .ClusterDomain }}
clusterDNS: [{{ .ClusterDNS }}]
authentication:
  x509:
    clientCAFile: {{ .KubeletClientsCAFile }}
  anonymous:
    enabled: true
rotateCertificates: true
nodeLeaseDurationSeconds: 20
tlsCipherSuites:
- TLS_CHACHA20_POLY1305_SHA256
- TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
- TLS_AES_128_GCM_SHA256
- TLS_AES_256_GCM_SHA384
- TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
featureGates:
`))

func NewGetBootstrapConfig(rt *api.Runtime) operations.GetBootstrapConfigHandler {
	return &getBootstrapConfig{rt}
}

type getBootstrapConfig struct {
	*api.Runtime
}

func (d *getBootstrapConfig) Handle(params operations.GetBootstrapConfigParams, principal *models.Principal) middleware.Responder {

	kluster, err := d.Klusters.Klusters(d.Namespace).Get(qualifiedName(params.Name, principal.Account))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 404, "Kluster not found")
		}
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "%s", err)
	}

	secret, err := util.KlusterSecret(d.Kubernetes, kluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 404, "Secret not found")
		}
		return NewErrorResponse(&operations.GetClusterCredentialsDefault{}, 500, "%s", err)
	}

	client, err := d.KlusterClientFactory.ClientFor(kluster)
	if err != nil {
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to create cluster client: %s", err)
	}

	token, tokenSecret, err := bootstraptoken.GenerateBootstrapToken(1 * time.Hour)
	if err != nil {
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to generate bootstrap token: %s", err)
	}
	if _, err := client.CoreV1().Secrets(tokenSecret.Namespace).Create(context.TODO(), tokenSecret, metav1.CreateOptions{}); err != nil {
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to store bootstrap token: %s", err)
	}

	kubeconfig := kubernetes.NewClientConfigV1(
		params.Name,
		"kubelet-bootstrap",
		kluster.Status.Apiserver,
		nil,
		nil,
		[]byte(secret.TLSCACertificate),
		token,
	)

	kubeconfigData, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to generate kubeconfig YAML document: %s", err)
	}

	kubeletConfig := struct {
		ClusterDomain        string
		ClusterDNS           string
		KubeletClientsCAFile string
	}{
		ClusterDomain:        kluster.Spec.DNSDomain,
		ClusterDNS:           kluster.Spec.DNSAddress,
		KubeletClientsCAFile: "/etc/kubernetes/certs/kubelet-clients-ca.pem",
	}
	var kubeletConfigYAML strings.Builder

	if ok, _ := util.KlusterVersionConstraint(kluster, ">= 1.17"); ok {
		if err := kubeletConfigurationTemplate117.Execute(&kubeletConfigYAML, kubeletConfig); err != nil {
			return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to generate kubelet config YAML document: %s", err)
		}
	} else {
		if err := kubeletConfigurationTemplate.Execute(&kubeletConfigYAML, kubeletConfig); err != nil {
			return NewErrorResponse(&operations.GetBootstrapConfigDefault{}, 500, "Failed to generate kubelet config YAML document: %s", err)
		}
	}

	credentials := models.BootstrapConfig{
		Kubeconfig:           string(kubeconfigData),
		KubeletClientsCA:     secret.KubeletClientsCACertificate,
		KubeletClientsCAFile: kubeletConfig.KubeletClientsCAFile,
		Config:               kubeletConfigYAML.String(),
	}

	return operations.NewGetBootstrapConfigOK().WithPayload(&credentials)
}
