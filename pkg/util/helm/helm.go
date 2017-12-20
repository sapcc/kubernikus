package helm

import (
	"fmt"
	"log"
	"net/url"

	yaml "gopkg.in/yaml.v2"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type OpenstackOptions struct {
	AuthURL    string
	Username   string
	Password   string
	DomainName string
	Region     string
}

type imageValues struct {
	Repostiory string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	PullPolicy string `yaml:"pullPolicy"`
}

type openstackValues struct {
	AuthURL             string `yaml:"authURL"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	DomainName          string `yaml:"domainName"`
	ProjectID           string `yaml:"projectID"`
	Region              string `yaml:"region"`
	LbSubnetID          string `yaml:"lbSubnetID"`
	LbFloatingNetworkID string `yaml:"lbFloatingNetworkID"`
	RouterID            string `yaml:"routerID"`
}

type persistenceValues struct {
	AccessMode string `yaml:"accessMode,omitempty"`
}

type etcdValues struct {
	Persistence persistenceValues `yaml:"persistence,omitempty"`
}

type apiValues struct {
	ApiserverHost string `yaml:"apiserverHost,omitempty"`
	WormholeHost  string `yaml:"wormholeHost,omitempty"`
}

type versionValues struct {
	Kubernetes string `yaml:"kubernetes,omitempty"`
	Kubernikus string `yaml:"kubernikus,omitempty"`
}

type kubernikusHelmValues struct {
	Image            imageValues       `yaml:"image,omitempty"`
	Openstack        openstackValues   `yaml:"openstack,omitempty"`
	Certs            map[string]string `yaml:"certs,omitempty"`
	ClusterCIDR      string            `yaml:"clusterCIDR,omitempty"`
	ServiceCIDR      string            `yaml:"serviceCIDR,omitempty"`
	AdvertiseAddress string            `yaml:"advertiseAddress,omitempty"`
	BoostrapToken    string            `yaml:"bootstrapToken,omitempty"`
	Version          versionValues     `yaml:"version,omitempty"`
	Etcd             etcdValues        `yaml:"etcd,omitempty"`
	Api              apiValues         `yaml:"api,omitempty"`
}

func KlusterToHelmValues(kluster *v1.Kluster, openstack *OpenstackOptions, certificates map[string]string, bootstrapToken string, accessMode string) ([]byte, error) {
	apiserverURL, err := url.Parse(kluster.Status.Apiserver)
	if err != nil {
		log.Fatal(err)
	}

	wormholeURL, err := url.Parse(kluster.Status.Wormhole)
	if err != nil {
		log.Fatal(err)
	}

	values := kubernikusHelmValues{
		Image:            getImageValues(kluster),
		BoostrapToken:    bootstrapToken,
		Certs:            certificates,
		ClusterCIDR:      kluster.Spec.ClusterCIDR,
		ServiceCIDR:      kluster.Spec.ServiceCIDR,
		AdvertiseAddress: kluster.Spec.AdvertiseAddress,
		Version: versionValues{
			Kubernetes: kluster.Spec.Version,
			Kubernikus: kluster.Status.Version,
		},
		Openstack: openstackValues{
			AuthURL:             openstack.AuthURL,
			Username:            openstack.Username,
			Password:            openstack.Password,
			DomainName:          openstack.DomainName,
			Region:              openstack.Region,
			ProjectID:           kluster.Spec.Openstack.ProjectID,
			LbSubnetID:          kluster.Spec.Openstack.LBSubnetID,
			LbFloatingNetworkID: kluster.Spec.Openstack.LBFloatingNetworkID,
			RouterID:            kluster.Spec.Openstack.RouterID,
		},
		Etcd: etcdValues{
			Persistence: persistenceValues{
				AccessMode: accessMode,
			},
		},
		Api: apiValues{
			ApiserverHost: apiserverURL.Hostname(),
			WormholeHost:  wormholeURL.Hostname(),
		},
	}

	result, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getImageValues(kluster *v1.Kluster) imageValues {
	v := imageValues{
		Repostiory: "quay.io/coreos/hyperkube",
		PullPolicy: "IfNotPresent",
	}

	if kluster.Spec.Version == "1.7.5" {
		v.Tag = "v1.7.5_coreos.1"
	} else {
		v.Tag = fmt.Sprintf("v%s_coreos.0", kluster.Spec.Version)
	}

	return v
}
