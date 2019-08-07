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

	"github.com/sapcc/kubernikus/pkg/api/models"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

type ignition struct {
}

var Ignition = &ignition{}

var passwordHashRounds = 1000000

func (i *ignition) getIgnitionTemplate(kluster *kubernikusv1.Kluster) (string, error) {
	switch {
	case strings.HasPrefix(kluster.Spec.Version, "1.15"):
		return Node_1_14, nil // No changes to 1.14
	case strings.HasPrefix(kluster.Spec.Version, "1.14"):
		return Node_1_14, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.13"):
		return Node_1_12, nil // No changes to 1.12
	case strings.HasPrefix(kluster.Spec.Version, "1.12"):
		return Node_1_12, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.11"):
		return Node_1_11, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.10"):
		return Node_1_10, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.9"):
		return Node_1_9, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.8"):
		return Node_1_8, nil
	case strings.HasPrefix(kluster.Spec.Version, "1.7"):
		return Node_1_7, nil
	default:
		return "", fmt.Errorf("Can't find iginition template for version %s", kluster.Spec.Version)
	}
}

func (i *ignition) GenerateNode(kluster *kubernikusv1.Kluster, pool *models.NodePool, nodeName string, secret *kubernikusv1.Secret, calicoNetworking bool, imageRegistry version.ImageRegistry, logger log.Logger) ([]byte, error) {

	ignition, err := i.getIgnitionTemplate(kluster)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("node").Funcs(sprig.TxtFuncMap()).Parse(ignition)
	if err != nil {
		return nil, err
	}

	//this is the old default for backwards comptibility with clusters that don't have a passwort generated
	//TODO: Remove once all klusters are upgraded
	passwordHash := "$6$rounds=1000000$aldshc,xbneroyw$I756LN/FtceE1deC2H.tGeSdeeelaaZWRwzmbEuO1SANf7ssyPjnbQjlW/FcMvWGUGrhF64tX9fK0abE/4oQ80"
	if secret.NodePassword != "" {
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
		passwordHash, err = passwordCrypter.Generate([]byte(secret.NodePassword), append([]byte(fmt.Sprintf("%srounds=%d$", sha512_crypt.MagicPrefix, passwordHashRounds)), salt...))
		if err != nil {
			return nil, fmt.Errorf("Faied to generate salted password: %s", err)
		}
	}
	var nodeLabels []string
	var nodeTaints []string
	if pool != nil {
		nodeLabels = append(nodeLabels, "ccloud.sap.com/nodepool="+pool.Name)
		if strings.HasPrefix(pool.Flavor, "zg") {
			nodeLabels = append(nodeLabels, "gpu=nvidia-tesla-v100")
		}
		if strings.HasPrefix(pool.Flavor, "zg") {
			nodeTaints = append(nodeTaints, "nvidia.com/gpu=present:NoSchedule")
		}
		for _, userTaint := range pool.Taints {
			nodeTaints = append(nodeTaints, userTaint)
		}
		for _, userLabel := range pool.Labels {
			nodeLabels = append(nodeLabels, userLabel)
		}
	}

	images, found := imageRegistry.Versions[kluster.Spec.Version]
	if !found {
		return nil, fmt.Errorf("Can't find images for version: %s ", kluster.Spec.Version)
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
		NodeLabels                         []string
		NodeTaints                         []string
		NodeName                           string
		HyperkubeImage                     string
		HyperkubeImageTag                  string
		CalicoNetworking                   bool
	}{
		TLSCA:                              secret.TLSCACertificate,
		KubeletClientsCA:                   secret.KubeletClientsCACertificate,
		ApiserverClientsSystemKubeProxy:    secret.ApiserverClientsKubeProxyCertificate,
		ApiserverClientsSystemKubeProxyKey: secret.ApiserverClientsKubeProxyPrivateKey,
		BootstrapToken:                     secret.BootstrapToken,
		ClusterCIDR:                        kluster.Spec.ClusterCIDR,
		ClusterDNSAddress:                  kluster.Spec.DNSAddress,
		ClusterDomain:                      kluster.Spec.DNSDomain,
		ApiserverURL:                       kluster.Status.Apiserver,
		ApiserverIP:                        kluster.Spec.AdvertiseAddress,
		OpenstackAuthURL:                   secret.Openstack.AuthURL,
		OpenstackUsername:                  secret.Openstack.Username,
		OpenstackPassword:                  secret.Openstack.Password,
		OpenstackDomain:                    secret.Openstack.DomainName,
		OpenstackRegion:                    secret.Openstack.Region,
		OpenstackLBSubnetID:                kluster.Spec.Openstack.LBSubnetID,
		OpenstackLBFloatingNetworkID:       kluster.Spec.Openstack.LBFloatingNetworkID,
		OpenstackRouterID:                  kluster.Spec.Openstack.RouterID,
		KubernikusImage:                    "sapcc/kubernikus",
		KubernikusImageTag:                 version.GitCommit,
		LoginPassword:                      passwordHash,
		LoginPublicKey:                     kluster.Spec.SSHPublicKey,
		NodeLabels:                         nodeLabels,
		NodeTaints:                         nodeTaints,
		NodeName:                           nodeName,
		HyperkubeImage:                     images.Hyperkube.Repository,
		HyperkubeImageTag:                  images.Hyperkube.Tag,
		CalicoNetworking:                   calicoNetworking,
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
