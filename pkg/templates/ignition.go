package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/coreos/container-linux-config-transpiler/config"
	"github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ignition struct {
}

var Ignition = &ignition{}

func (i *ignition) GenerateNode(kluster *v1.Kluster, client kubernetes.Interface) ([]byte, error) {
	secret, err := client.CoreV1().Secrets(kluster.Namespace).Get(kluster.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("node").Funcs(sprig.TxtFuncMap()).Parse(Node)
	if err != nil {
		return nil, err
	}

	data := struct {
		TLSCA                              string
		KubeletClientsCA                   string
		ApiserverClientsSystemKubeProxy    string
		ApiserverClientsSystemKubeProxyKey string
		ClusterCIDR                        string
		ApiserverURL                       string
		BootstrapToken                     string
		OpenstackAuthURL                   string
		OpenstackUsername                  string
		OpenstackPassword                  string
		OpenstackDomain                    string
		OpenstackRegion                    string
		OpenstackLBSubnetID                string
		OpenstackRouterID                  string
		KubernikusImage                    string
		KubernikusImageTag                 string
	}{
		TLSCA:                              string(secret.Data["tls-ca.pem"]),
		KubeletClientsCA:                   string(secret.Data["kubelet-clients-ca.pem"]),
		ApiserverClientsSystemKubeProxy:    string(secret.Data["apiserver-clients-system-kube-proxy.pem"]),
		ApiserverClientsSystemKubeProxyKey: string(secret.Data["apiserver-clients-system-kube-proxy-key.pem"]),
		ClusterCIDR:                        "198.19.0.0/16",
		ApiserverURL:                       kluster.Spec.KubernikusInfo.ServerURL,
		BootstrapToken:                     kluster.Spec.KubernikusInfo.BootstrapToken,
		OpenstackAuthURL:                   kluster.Spec.OpenstackInfo.AuthURL,
		OpenstackUsername:                  kluster.Spec.OpenstackInfo.Username,
		OpenstackPassword:                  kluster.Spec.OpenstackInfo.Password,
		OpenstackDomain:                    kluster.Spec.OpenstackInfo.Domain,
		OpenstackRegion:                    kluster.Spec.OpenstackInfo.Region,
		OpenstackLBSubnetID:                kluster.Spec.OpenstackInfo.LBSubnetID,
		OpenstackRouterID:                  kluster.Spec.OpenstackInfo.RouterID,
		KubernikusImage:                    "sapcc/kubernikus",
		KubernikusImageTag:                 version.VERSION,
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, data)
	if err != nil {
		return nil, err
	}

	glog.V(6).Infof("IgnitionData: %v", data)
	glog.V(6).Infof("IgnitionYAML: %v", string(buffer.Bytes()))

	ignitionConfig, ast, report := config.Parse(buffer.Bytes())
	if len(report.Entries) > 0 {
		glog.V(2).Infof("Something odd while transpiling ignition file: %v", report.String())
		if report.IsFatal() {
			return nil, fmt.Errorf("Couldn't transpile ignition file: %v", report.String())
		}
	}

	ignitionConfig2_0, report := config.ConvertAs2_0(ignitionConfig, platform.OpenStackMetadata, ast)
	if len(report.Entries) > 0 {
		glog.V(2).Infof("Something odd while convertion ignition config: %v", report.String())
		if report.IsFatal() {
			return nil, fmt.Errorf("Couldn't convert ignition config: %v", report.String())
		}
	}

	var dataOut []byte
	dataOut, err = json.MarshalIndent(&ignitionConfig2_0, "", "  ")
	dataOut = append(dataOut, '\n')

	glog.V(6).Infof("IgnitionJSON: %v", string(dataOut))

	if err != nil {
		return nil, err
	}

	return dataOut, nil
}
