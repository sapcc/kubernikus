package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/aokoli/goutils"
	"github.com/coreos/container-linux-config-transpiler/config"
	"github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/go-kit/kit/log"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	"k8s.io/api/core/v1"

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

var passwordHashRounds = 1000000

func (i *ignition) getIgnitionTemplate(kluster *kubernikusv1.Kluster) string {
	switch {
	case strings.HasPrefix(kluster.Spec.Version, "1.10"):
		return Node_1_10
	case strings.HasPrefix(kluster.Spec.Version, "1.9"):
		return Node_1_9
	case strings.HasPrefix(kluster.Spec.Version, "1.8"):
		return Node_1_8
	default:
		return Node_1_7
	}
}

func (i *ignition) getIgnitionBareMetalTemplate(kluster *kubernikusv1.Kluster) string {
	switch {
	case strings.HasPrefix(kluster.Spec.Version, "1.10"):
		return BareMetalNode_1_10
	default:
		return BareMetalNode_1_10
	}
}

func (i *ignition) GenerateNode(kluster *kubernikusv1.Kluster, nodeName string, secret *v1.Secret, externalNode *kubernikusv1.ExternalNode, logger log.Logger) ([]byte, error) {
	for _, field := range i.requiredNodeSecrets {
		if _, ok := secret.Data[field]; !ok {
			return nil, fmt.Errorf("Field %s missing in secret", field)
		}
	}

	ignition := ""

	if externalNode == nil {
		externalNode = &kubernikusv1.ExternalNode{}
		ignition = i.getIgnitionTemplate(kluster)
	} else {
		ignition = i.getIgnitionBareMetalTemplate(kluster)
	}

	tmpl, err := template.New("node").Funcs(sprig.TxtFuncMap()).Parse(ignition)
	if err != nil {
		return nil, err
	}

	//this is the old default for backwards comptibility with clusters that don't have a passwort generated
	//TODO: Remove once all klusters are upgraded
	passwordHash := "$6$rounds=1000000$aldshc,xbneroyw$I756LN/FtceE1deC2H.tGeSdeeelaaZWRwzmbEuO1SANf7ssyPjnbQjlW/FcMvWGUGrhF64tX9fK0abE/4oQ80"
	if nodePassword, ok := secret.Data["node-password"]; ok {
		passwordCrypter := sha512_crypt.New()
		//generate 16 byte random salt
		salt, err := goutils.Random(sha512_crypt.SaltLenMax, 32, 127, true, true)
		if err != nil {
			return nil, fmt.Errorf("Unable to generate random salt: %s", err)
		}
		//We crank up the heat to 1 million rounds of hashing for this password (default 5000)
		//Reason for this is we expose the resulting hash in the metadata service which is not very secure.
		//It takes about 500ms on my workstation to compute this hash. So this means login to a node is also
		// delayed for about a second which should be ok as this password is only meant as a last resort.
		passwordHash, err = passwordCrypter.Generate(nodePassword, append([]byte(fmt.Sprintf("%srounds=%d$", sha512_crypt.MagicPrefix, passwordHashRounds)), salt...))
		if err != nil {
			return nil, fmt.Errorf("Faied to generate salted password: %s", err)
		}
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
		LoginPassword                      string
		LoginPublicKey                     string
		NodeName                           string
		ExternalNode                       *kubernikusv1.ExternalNode
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
		LoginPassword:                      passwordHash,
		LoginPublicKey:                     kluster.Spec.SSHPublicKey,
		NodeName:                           nodeName,
		ExternalNode:                       externalNode,
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
