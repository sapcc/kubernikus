package helm

import (
	"fmt"
	"hash/crc64"
	"math/rand"
	"net/url"

	"github.com/go-openapi/swag"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v2"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
	"github.com/sapcc/kubernikus/pkg/version"
)

//contains unamibious characters for generic random passwords
var randomPasswordChars = []rune("abcdefghjkmnpqrstuvwxABCDEFGHJKLMNPQRSTUVWX23456789")

var ETCDBackupAnnotation = "kubernikus.cloud.sap/backup"

var crc64ISOTable = crc64.MakeTable(crc64.ISO)

type OpenstackOptions struct {
	AuthURL    string
	Username   string
	Password   string
	DomainName string
	Region     string
}

type openstackValues struct {
	AuthURL             string `yaml:"authURL"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	DomainName          string `yaml:"domainName"`
	ProjectID           string `yaml:"projectID"`
	ProjectDomainName   string `yaml:"projectDomainName"`
	Region              string `yaml:"region,omitempty"`
	LbSubnetID          string `yaml:"lbSubnetID,omitempty"`
	LbFloatingNetworkID string `yaml:"lbFloatingNetworkID,omitempty"`
	RouterID            string `yaml:"routerID,omitempty"`
}

type persistenceValues struct {
	AccessMode string `yaml:"accessMode,omitempty"`
}

type etcdValues struct {
	Persistence      persistenceValues `yaml:"persistence,omitempty"`
	StorageContainer string            `yaml:"storageContainer,omitempty"`
	Openstack        openstackValues   `yaml:"openstack,omitempty"`
	Backup           etcdBackupValues  `yaml:"backup"`
}

type etcdBackupValues struct {
	Schedule        string `yaml:"schedule,omitempty"`
	Enabled         bool   `yaml:"enabled"`
	StorageProvider string `yaml:"storageProvider,omitempty"`
}

type apiValues struct {
	ApiserverHost string `yaml:"apiserverHost,omitempty"`
	WormholeHost  string `yaml:"wormholeHost,omitempty"`
}

type versionValues struct {
	Kubernetes string `yaml:"kubernetes,omitempty"`
	Kubernikus string `yaml:"kubernikus,omitempty"`
}

type dashboardValues struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

type dexValues struct {
	Enabled            bool           `yaml:"enabled,omitempty"`
	StaticClientSecret string         `yaml:"staticClientSecret,omitempty"`
	StaticPassword     staticPassword `yaml:"staticPasword,omitempty"`
	Connectors         dexConnectors  `yaml:"connectors,omitempty"`
}

type dexConnectors struct {
	Keystone dexKeystoneConnector `yaml:"keystone"`
	LDAP     dexLDAPConnector     `yaml:"ldap"`
}

type dexKeystoneConnector struct {
	Enabled bool `yaml:"enabled"`
}
type dexLDAPConnector struct {
	Enabled bool `yaml:"enabled"`
}

type staticPassword struct {
	HashedPassword string `yaml:"hashedPassword,omitempty"`
}

type kubernikusHelmValues struct {
	Openstack        openstackValues       `yaml:"openstack,omitempty"`
	ClusterCIDR      string                `yaml:"clusterCIDR,omitempty"`
	ServiceCIDR      string                `yaml:"serviceCIDR,omitempty"`
	AdvertiseAddress string                `yaml:"advertiseAddress,omitempty"`
	AdvertisePort    int64                 `yaml:"advertisePort,omitempty"`
	BoostrapToken    string                `yaml:"bootstrapToken,omitempty"`
	Version          versionValues         `yaml:"version,omitempty"`
	Etcd             etcdValues            `yaml:"etcd,omitempty"`
	Api              apiValues             `yaml:"api,omitempty"`
	Name             string                `yaml:"name"`
	Account          string                `yaml:"account"`
	SecretName       string                `yaml:"secretName"`
	ImageRegistry    version.ImageRegistry `yaml:",inline"`
	Dex              dexValues             `yaml:"dex,omitempty"`
	Dashboard        dashboardValues       `yaml:"dashboard,omitempty"`
}

func KlusterToHelmValues(kluster *v1.Kluster, secret *v1.Secret, kubernetesVersion string, registry *version.ImageRegistry, accessMode string) ([]byte, error) {
	apiserverURL, err := url.Parse(kluster.Status.Apiserver)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse apiserver URL: %s", err)
	}

	wormholeURL, err := url.Parse(kluster.Status.Wormhole)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse wormhole server URL: %s", err)
	}

	//Get a deterministic value for the cluster between 0-59 for the hourly etcd full backup schedule
	//calculate a crc64 checksum of the kluster UID
	uidChecksum := crc64.Checksum([]byte(kluster.UID), crc64ISOTable)
	backupMinute := rand.New(rand.NewSource(int64(uidChecksum))).Intn(60)

	hashedPassword := ""

	if swag.BoolValue(kluster.Spec.Dex) {

		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(secret.DexStaticPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("Failed to hash dex static password: %v", err)
		}
		hashedPassword = string(hashedBytes)
	}

	dex := dexValues{
		Enabled: swag.BoolValue(kluster.Spec.Dex),
		StaticPassword: staticPassword{
			HashedPassword: hashedPassword,
		},
		StaticClientSecret: secret.DexClientSecret,
		Connectors: dexConnectors{
			Keystone: dexKeystoneConnector{Enabled: !kluster.Spec.NoCloud},
			LDAP:     dexLDAPConnector{Enabled: kluster.Spec.NoCloud},
		},
	}

	values := kubernikusHelmValues{
		Account:          kluster.Account(),
		BoostrapToken:    secret.BootstrapToken,
		ClusterCIDR:      kluster.ClusterCIDR(),
		SecretName:       kluster.Name + "-secret",
		ServiceCIDR:      kluster.Spec.ServiceCIDR,
		AdvertiseAddress: kluster.Spec.AdvertiseAddress,
		AdvertisePort:    kluster.Spec.AdvertisePort,
		Name:             kluster.Spec.Name,
		Version: versionValues{
			Kubernetes: kubernetesVersion,
			Kubernikus: version.GitCommit,
		},
		Etcd: etcdValues{
			Backup: etcdBackupValues{
				Enabled:  kluster.Spec.Backup != "off",
				Schedule: fmt.Sprintf("%d * * * *", backupMinute),
				// Default storage provider is Swift, add more providers here
				StorageProvider: func(backupType string) string {
					if backupType == "externalAWS" {
						return "S3"
					}
					return "Swift"
				}(kluster.Spec.Backup),
			},
			Persistence: persistenceValues{
				AccessMode: accessMode,
			},
			StorageContainer: etcd_util.DefaultStorageContainer(kluster),
			Openstack: openstackValues{
				AuthURL:           secret.Openstack.AuthURL,
				Username:          secret.Openstack.Username,
				Password:          secret.Openstack.Password,
				DomainName:        secret.Openstack.DomainName,
				ProjectID:         secret.Openstack.ProjectID,
				ProjectDomainName: secret.Openstack.ProjectDomainName,
			},
		},
		Api: apiValues{
			ApiserverHost: apiserverURL.Hostname(),
			WormholeHost:  wormholeURL.Hostname(),
		},
		Dashboard: dashboardValues{
			Enabled: swag.BoolValue(kluster.Spec.Dashboard),
		},
		Dex: dex,
	}
	if !kluster.Spec.NoCloud {
		values.Openstack = openstackValues{
			AuthURL:             secret.Openstack.AuthURL,
			Username:            secret.Openstack.Username,
			Password:            secret.Openstack.Password,
			DomainName:          secret.Openstack.DomainName,
			Region:              secret.Openstack.Region,
			ProjectID:           kluster.Spec.Openstack.ProjectID,
			ProjectDomainName:   secret.ProjectDomainName,
			LbSubnetID:          kluster.Spec.Openstack.LBSubnetID,
			LbFloatingNetworkID: kluster.Spec.Openstack.LBFloatingNetworkID,
			RouterID:            kluster.Spec.Openstack.RouterID,
		}
	}
	if registry != nil {
		values.ImageRegistry = *registry
	}

	result, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	//Temporary unmarshal to map[string]interface{}
	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(result, &m)
	if err != nil {
		return nil, err
	}
	// Merge extra values via deep merge
	r := MergeMaps(m, secret.ExtraValues)
	// Remarshal to byte[]
	result, err = yaml.Marshal(r)
	if err != nil {
		return nil, err
	}
	return result, nil
}
