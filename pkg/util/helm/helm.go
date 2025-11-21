package helm

import (
	"fmt"
	"hash/crc64"
	"math/rand"
	"net/url"

	"github.com/go-openapi/swag/conv"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/yaml"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
	"github.com/sapcc/kubernikus/pkg/version"
)

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
	AuthURL             string `yaml:"authURL,omitempty" json:"authURL,omitempty"`
	Username            string `yaml:"username,omitempty" json:"username,omitempty"`
	Password            string `yaml:"password,omitempty" json:"password,omitempty"`
	DomainName          string `yaml:"domainName,omitempty" json:"domainName,omitempty"`
	ProjectID           string `yaml:"projectID,omitempty" json:"projectID,omitempty"`
	ProjectDomainName   string `yaml:"projectDomainName,omitempty" json:"projectDomainName,omitempty"`
	Region              string `yaml:"region,omitempty" json:"region,omitempty"`
	LbSubnetID          string `yaml:"lbSubnetID,omitempty" json:"lbSubnetID,omitempty"`
	LbFloatingNetworkID string `yaml:"lbFloatingNetworkID,omitempty" json:"lbFloatingNetworkID,omitempty"`
	RouterID            string `yaml:"routerID,omitempty" json:"routerID,omitempty"`
	UseOctavia          bool   `yaml:"useOctavia,omitempty" json:"useOctavia,omitempty"`
}

type persistenceValues struct {
	AccessMode string `yaml:"accessMode,omitempty" json:"accessMode,omitempty"`
}

type etcdValues struct {
	Persistence      persistenceValues      `yaml:"persistence,omitempty" json:"persistence,omitempty"`
	StorageContainer string                 `yaml:"storageContainer,omitempty" json:"storageContainer,omitempty"`
	Openstack        openstackValues        `yaml:"openstack,omitempty" json:"openstack,omitempty"`
	Backup           etcdBackupValues       `yaml:"backup" json:"backup"`
	Images           version.KlusterVersion `yaml:"images" json:"images"`
	Version          versionValues          `yaml:"version,omitempty" json:"version,omitempty"`
}

type etcdBackupValues struct {
	Schedule        string `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	StorageProvider string `yaml:"storageProvider,omitempty" json:"storageProvider,omitempty"`
}

type apiValues struct {
	ApiserverHost      string     `yaml:"apiserverHost,omitempty" json:"apiserverHost,omitempty"`
	WormholeHost       string     `yaml:"wormholeHost,omitempty" json:"wormholeHost,omitempty"`
	CORSAllowedOrigins string     `yaml:"corsAllowedOrigins,omitempty" json:"corsAllowedOrigins,omitempty"`
	SNICertSecret      string     `yaml:"sniCertSecret,omitempty" json:"sniCertSecret,omitempty"`
	OIDC               oidcValues `yaml:"oidc,omitempty" json:"oidc,omitempty"`
}

type versionValues struct {
	Kubernetes string `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`
	Kubernikus string `yaml:"kubernikus,omitempty" json:"kubernikus,omitempty"`
}

type dashboardValues struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

type dexValues struct {
	Enabled            bool           `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	StaticClientSecret string         `yaml:"staticClientSecret,omitempty" json:"staticClientSecret,omitempty"`
	StaticPassword     staticPassword `yaml:"staticPasword,omitempty" json:"staticPasword,omitempty"`
	Connectors         dexConnectors  `yaml:"connectors,omitempty" json:"connectors,omitempty"`
}

type oidcValues struct {
	IssuerURL string `yaml:"issuerURL,omitempty" json:"issuerURL,omitempty"`
	ClientID  string `yaml:"clientID,omitempty" json:"clientID,omitempty"`
}

type dexConnectors struct {
	Keystone dexKeystoneConnector `yaml:"keystone" json:"keystone"`
	LDAP     dexLDAPConnector     `yaml:"ldap" json:"ldap"`
}

type dexKeystoneConnector struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}
type dexLDAPConnector struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type staticPassword struct {
	HashedPassword string `yaml:"hashedPassword,omitempty" json:"hashedPassword,omitempty"`
}

type kubernikusHelmValues struct {
	Openstack        openstackValues        `yaml:"openstack,omitempty" json:"openstack,omitempty"`
	Audit            string                 `yaml:"audit" json:"audit"`
	ClusterCIDR      string                 `yaml:"clusterCIDR,omitempty" json:"clusterCIDR,omitempty"`
	ServiceCIDR      string                 `yaml:"serviceCIDR,omitempty" json:"serviceCIDR,omitempty"`
	AdvertiseAddress string                 `yaml:"advertiseAddress,omitempty" json:"advertiseAddress,omitempty"`
	AdvertisePort    int64                  `yaml:"advertisePort,omitempty" json:"advertisePort,omitempty"`
	BootstrapToken   string                 `yaml:"bootstrapToken,omitempty" json:"bootstrapToken,omitempty"`
	Version          versionValues          `yaml:"version,omitempty" json:"version,omitempty"`
	Etcd             etcdValues             `yaml:"etcd,omitempty" json:"etcd,omitempty"`
	Api              apiValues              `yaml:"api,omitempty" json:"api,omitempty"`
	Name             string                 `yaml:"name" json:"name"`
	Account          string                 `yaml:"account" json:"account"`
	SecretName       string                 `yaml:"secretName" json:"secretName"`
	Images           version.KlusterVersion `yaml:"images" json:"images"`
	Dex              dexValues              `yaml:"dex,omitempty" json:"dex,omitempty"`
	Dashboard        dashboardValues        `yaml:"dashboard,omitempty" json:"dashboard,omitempty"`
}

func KlusterToHelmValues(kluster *v1.Kluster, secret *v1.Secret, kubernetesVersion string, registry *version.ImageRegistry, accessMode string) (map[string]interface{}, error) {
	apiserverURL, err := url.Parse(kluster.Status.Apiserver)
	if err != nil {
		return nil, fmt.Errorf("failed to parse apiserver URL: %s", err)
	}

	wormholeURL, err := url.Parse(kluster.Status.Wormhole)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wormhole server URL: %s", err)
	}

	//Get a deterministic value for the cluster between 0-59 for the hourly etcd full backup schedule
	//calculate a crc64 checksum of the kluster UID
	uidChecksum := crc64.Checksum([]byte(kluster.UID), crc64ISOTable)
	backupMinute := rand.New(rand.NewSource(int64(uidChecksum))).Intn(60)

	hashedPassword := ""

	if conv.Value(kluster.Spec.Dex) {

		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(secret.DexStaticPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash dex static password: %v", err)
		}
		hashedPassword = string(hashedBytes)
	}

	dex := dexValues{
		Enabled: conv.Value(kluster.Spec.Dex),
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
		BootstrapToken:   secret.BootstrapToken,
		Audit:            conv.Value(kluster.Spec.Audit),
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
				AuthURL:           secret.AuthURL,
				Username:          secret.Username,
				Password:          secret.Password,
				DomainName:        secret.DomainName,
				ProjectID:         secret.ProjectID,
				ProjectDomainName: secret.ProjectDomainName,
			},
			Version: versionValues{
				Kubernetes: kubernetesVersion,
				Kubernikus: version.GitCommit,
			},
		},
		Api: apiValues{
			ApiserverHost: apiserverURL.Hostname(),
			WormholeHost:  wormholeURL.Hostname(),
		},
		Dashboard: dashboardValues{
			Enabled: conv.Value(kluster.Spec.Dashboard),
		},
		Dex: dex,
	}
	if registry != nil {
		values.Images = registry.Versions[kubernetesVersion]
		// make etcd images available to subchart
		values.Etcd.Images.Etcd = values.Images.Etcd
		values.Etcd.Images.EtcdBackup = values.Images.EtcdBackup
	}
	if !kluster.Spec.NoCloud {
		values.Openstack = openstackValues{
			AuthURL:             secret.AuthURL,
			Username:            secret.Username,
			Password:            secret.Password,
			DomainName:          secret.DomainName,
			Region:              secret.Region,
			ProjectID:           kluster.Account(),
			ProjectDomainName:   secret.ProjectDomainName,
			LbSubnetID:          kluster.Spec.Openstack.LBSubnetID,
			LbFloatingNetworkID: kluster.Spec.Openstack.LBFloatingNetworkID,
			RouterID:            kluster.Spec.Openstack.RouterID,
			UseOctavia:          true,
		}
	}
	if kluster.Spec.Oidc != nil {
		values.Api.OIDC = oidcValues{
			IssuerURL: kluster.Spec.Oidc.IssuerURL,
			ClientID:  kluster.Spec.Oidc.ClientID,
		}
	}

	result, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	// Unmarshal string inside ExtraValues into map
	extraValues := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(secret.ExtraValues), &extraValues)
	if err != nil {
		return nil, err
	}
	// Temporary unmarshal values as well
	m := make(map[string]interface{})
	err = yaml.Unmarshal(result, &m)
	if err != nil {
		return nil, err
	}
	// Merge extra values via deep merge
	r := MergeMaps(m, extraValues)
	return r, nil
}
