package v1

import (
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)



type NodePoolConfig struct {
	Upgrade bool `json:"upgrade"`
	Repair  bool `json:"repair"`
}

type NodePool struct {
	Name   string         `json:"name"`
	Size   int            `json:"size"`
	Flavor string         `json:"flavor"`
	Image  string         `json:"image"`
	Config NodePoolConfig `json:"config"`
}

type OpenstackSpec struct {
	AuthURL    string `json:"authURL"`
	ProjectID  string `json:"projectID"`
	RouterID   string `json:"routerID"`
	NetworkID  string `json:"networkID"`
	LBSubnetID string `json:"lbSubnetID"`
	Domain     string `json:"domain"`
	Region     string `json:"region"`
	Username   string `json:"username"`
}

type KlusterSpec struct {
	Name             string        `json:"name"`
	Domain           string        `json:"domain"`
	ClusterCIDR      string        `json:"clusterCIDR"`
	ClusterDNS       string        `json:"clusterDNS"`
	ClusterDNSDomain string        `json:"clusterDNSDomain"`
	ServiceCIDR      string        `json:"serviceCIDR"`
	AdvertiseAddress string        `json:"advertiseAddress"`
	NodePools        []NodePool    `json:"nodePools,omitempty"`
	Openstack        OpenstackSpec `json:"openstack,omitempty"`
}

type KlusterState string

const (
	KlusterPending     KlusterState = "Pending"
	KlusterCreating    KlusterState = "Creating"
	KlusterReady       KlusterState = "Ready"
	KlusterTerminating KlusterState = "Terminating"
	KlusterTerminated  KlusterState = "Terminated"
	KlusterError       KlusterState = "Error"
)

type KlusterStatus struct {
	Kluster   KlusterInfo    `json:"kluster"`
	Apiserver string         `json:"apiserver"`
	Wormhole  string         `json:"wormhole"`
	NodePools []NodePoolInfo `json:"nodePools,omitempty"`
}

type KlusterInfo struct {
	State   KlusterState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type OpenstackSecret struct {
	Password string `json:"password"`
}

type KlusterSecret struct {
	BootstrapToken string            `json:"bootstrapToken"`
	Certificates   map[string]string `json:"certificates"`
	Openstack      OpenstackSecret   `json:"openstack,omitempty"`
}

type NodePoolInfo struct {
	Name        string `json:"name"`
	Size        int    `json:size`
	Running     int    `json:running`
	Healthy     int    `json:healthy`
	Schedulable int    `json:schedulable`
}

// +genclient

type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              KlusterSpec   `json:"spec"`
	Status            KlusterStatus `json:"status,omitempty"`
	Secret            KlusterSecret `json:"secret"`
}

type KlusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Kluster `json:"items"`
}

func (spec KlusterSpec) Validate() error {
	//Add some validation
	return nil
}

func (spec Kluster) Account() string {
	return spec.ObjectMeta.Labels["account"]
}

func (kluster Kluster) ToHelmValues() ([]byte, error) {
	type Values struct {
		Kubernikus struct {
			BoostrapToken string `yaml:"bootstrapToken,omitempty"`
		}
		Openstack struct {
			AuthURL    string `yaml:"authURL"`
			Username   string `yaml:"username"`
			Password   string `yaml:"password"`
			DomainName string `yaml:"domainName"`
			ProjectID  string `yaml:"projectID"`
			Region     string `yaml:"Region"`
			LbSubnetID string `yaml:"lbSubnetID"`
			RouterID   string `yaml:"routerID"`
		}
		Certs            map[string]string `yaml:"certs,omitempty"`
		ClusterCIDR      string            `yaml:"clusterCIDR,omitempty"`
		ServiceCIDR      string            `yaml:"serviceCIDR,omitempty"`
		AdvertiseAddress string            `yaml:"advertiseAddress,omitempty"`
	}

	values := Values{}
	values.Kubernikus.BoostrapToken = kluster.Secret.BootstrapToken
	values.Openstack.AuthURL = kluster.Spec.Openstack.AuthURL
	values.Openstack.Username = kluster.Spec.Openstack.Username
	values.Openstack.Password = kluster.Secret.Openstack.Password
	values.Openstack.DomainName = kluster.Spec.Openstack.Domain
	values.Openstack.ProjectID = kluster.Spec.Openstack.ProjectID
	values.Openstack.Region = kluster.Spec.Openstack.Region
	values.Openstack.LbSubnetID = kluster.Spec.Openstack.LBSubnetID
	values.Openstack.RouterID = kluster.Spec.Openstack.RouterID
	values.Certs = kluster.Secret.Certificates
	values.ClusterCIDR = kluster.Spec.ClusterCIDR
	values.ServiceCIDR = kluster.Spec.ServiceCIDR
	values.AdvertiseAddress = kluster.Spec.AdvertiseAddress

	result, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	return result, nil
}
