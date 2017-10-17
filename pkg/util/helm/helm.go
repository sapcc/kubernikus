package helm

import (
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	yaml "gopkg.in/yaml.v2"
)

type OpenstackOptions struct {
	AuthURL    string
	Username   string
	Password   string
	DomainName string
	Region     string
}

type openstackValues struct {
	AuthURL    string `yaml:"authURL"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	DomainName string `yaml:"domainName"`
	ProjectID  string `yaml:"projectID"`
	Region     string `yaml:"Region"`
	LbSubnetID string `yaml:"lbSubnetID"`
	RouterID   string `yaml:"routerID"`
}

type persistenceValues struct {
	AccessMode string `yaml:"accessMode,omitempty"`
}

type etcdValues struct {
	Persistence persistenceValues `yaml:"persistence,omitempty"`
}

type kubernikusHelmValues struct {
	Openstack        openstackValues   `yaml:"openstack,omitempty"`
	Certs            map[string]string `yaml:"certs,omitempty"`
	ClusterCIDR      string            `yaml:"clusterCIDR,omitempty"`
	ServiceCIDR      string            `yaml:"serviceCIDR,omitempty"`
	AdvertiseAddress string            `yaml:"advertiseAddress,omitempty"`
	BoostrapToken    string            `yaml:"bootstrapToken,omitempty"`
	Etcd             etcdValues        `yaml:"etcd,omitempty"`
}

func KlusterToHelmValues(kluster *v1.Kluster, openstack *OpenstackOptions, certificates map[string]string, bootstrapToken string, accessMode string) ([]byte, error) {
	values := kubernikusHelmValues{
		BoostrapToken:    bootstrapToken,
		Certs:            certificates,
		ClusterCIDR:      kluster.Spec.ClusterCIDR,
		ServiceCIDR:      kluster.Spec.ServiceCIDR,
		AdvertiseAddress: kluster.Spec.AdvertiseAddress,
		Openstack: openstackValues{
			AuthURL:    openstack.AuthURL,
			Username:   openstack.Username,
			Password:   openstack.Password,
			DomainName: openstack.DomainName,
			Region:     openstack.Region,
			ProjectID:  kluster.Spec.Openstack.ProjectID,
			LbSubnetID: kluster.Spec.Openstack.LBSubnetID,
			RouterID:   kluster.Spec.Openstack.RouterID,
		},
		Etcd: etcdValues{
			Persistence: persistenceValues{
				AccessMode: accessMode,
			},
		},
	}

	result, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	return result, nil
}
