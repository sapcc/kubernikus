package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/coreos/container-linux-config-transpiler/config"
	"github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/pkg/api/v1"

	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

type ignition struct {
	requiredNodeSecrets []string
}

var Ignition = &ignition{
	requiredNodeSecrets: []string{
		"tls-ca.pem",
		"kubelet-clients-ca.pem",
		"apiserver-clients-system-kube-proxy.pem",
		"apiserver-clients-system-kube-proxy-key.pem",
		"bootstrapToken",
		"openstack-auth-url",
		"openstack-username",
		"openstack-password",
		"openstack-domain-name",
		"openstack-region",
	},
}

func (i *ignition) getIgnitionTemplate(kluster *kubernikusv1.Kluster) string {
	switch {
	case strings.HasPrefix(kluster.Spec.Version, "1.9"):
		return Node_1_9
	case strings.HasPrefix(kluster.Spec.Version, "1.8"):
		return Node_1_8
	default:
		return Node_1_7
	}
}

func (i *ignition) GenerateNode(kluster *kubernikusv1.Kluster, secret *v1.Secret, logger log.Logger) ([]byte, error) {
	for _, field := range i.requiredNodeSecrets {
		if _, ok := secret.Data[field]; !ok {
			return nil, fmt.Errorf("Field %s missing in secret", field)
		}
	}

	ignition := i.getIgnitionTemplate(kluster)
	tmpl, err := template.New("node").Funcs(sprig.TxtFuncMap()).Parse(ignition)
	if err != nil {
		return nil, err
	}

	data := struct {
		TLSCA                              string
		KubeletClientsCA                   string
		ApiserverClientsSystemKubeProxy    string
		ApiserverClientsSystemKubeProxyKey string
		ClusterDomain                      string
		ClusterDNSAddress                  string
		ClusterCIDR                        string
		ApiserverURL                       string
		ApiserverIP                        string
		BootstrapToken                     string
		OpenstackAuthURL                   string
		OpenstackUsername                  string
		OpenstackPassword                  string
		OpenstackDomain                    string
		OpenstackRegion                    string
		OpenstackLBSubnetID                string
		OpenstackLBFloatingNetworkID       string
		OpenstackRouterID                  string
		KubernikusImage                    string
		KubernikusImageTag                 string
	}{
		TLSCA:                              string(secret.Data["tls-ca.pem"]),
		KubeletClientsCA:                   string(secret.Data["kubelet-clients-ca.pem"]),
		ApiserverClientsSystemKubeProxy:    string(secret.Data["apiserver-clients-system-kube-proxy.pem"]),
		ApiserverClientsSystemKubeProxyKey: string(secret.Data["apiserver-clients-system-kube-proxy-key.pem"]),
		BootstrapToken:                     string(secret.Data["bootstrapToken"]),
		ClusterCIDR:                        kluster.Spec.ClusterCIDR,
		ClusterDNSAddress:                  kluster.Spec.DNSAddress,
		ClusterDomain:                      kluster.Spec.DNSDomain,
		ApiserverURL:                       kluster.Status.Apiserver,
		ApiserverIP:                        kluster.Spec.AdvertiseAddress,
		OpenstackAuthURL:                   string(secret.Data["openstack-auth-url"]),
		OpenstackUsername:                  string(secret.Data["openstack-username"]),
		OpenstackPassword:                  string(secret.Data["openstack-password"]),
		OpenstackDomain:                    string(secret.Data["openstack-domain-name"]),
		OpenstackRegion:                    string(secret.Data["openstack-region"]),
		OpenstackLBSubnetID:                kluster.Spec.Openstack.LBSubnetID,
		OpenstackLBFloatingNetworkID:       kluster.Spec.Openstack.LBFloatingNetworkID,
		OpenstackRouterID:                  kluster.Spec.Openstack.RouterID,
		KubernikusImage:                    "sapcc/kubernikus",
		KubernikusImageTag:                 version.GitCommit,
	}

	var dataOut []byte
	var buffer bytes.Buffer
	var report report.Report

	defer func() {
		logger.Log(
			"msg", "ignition debug",
			"data", data,
			"yaml", string(buffer.Bytes()),
			"json", string(dataOut),
			"report", report.String(),
			"v", 6,
			"err", err)
	}()

	err = tmpl.Execute(&buffer, data)
	if err != nil {
		return nil, err
	}

	ignitionConfig, ast, report := config.Parse(buffer.Bytes())
	if len(report.Entries) > 0 {
		if report.IsFatal() {
			return nil, fmt.Errorf("Couldn't transpile ignition file: %v", report.String())
		}
	}

	ignitionConfig2_0, report := config.ConvertAs2_0(ignitionConfig, platform.OpenStackMetadata, ast)
	if len(report.Entries) > 0 {
		if report.IsFatal() {
			return nil, fmt.Errorf("Couldn't convert ignition config: %v", report.String())
		}
	}

	dataOut, err = json.MarshalIndent(&ignitionConfig2_0, "", "  ")
	dataOut = append(dataOut, '\n')

	if err != nil {
		return nil, err
	}

	return dataOut, nil
}
