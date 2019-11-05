package kubernikus

import (
	"fmt"
	"net"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
	"github.com/sapcc/kubernikus/pkg/migration"
	"github.com/sapcc/kubernikus/pkg/util/ip"
	"github.com/sapcc/kubernikus/pkg/version"
)

var (
	//Keep this in sync with the default in swagger.yaml
	DEFAULT_CLUSTER_CIDR      = spec.MustDefaultString("KlusterSpec", "clusterCIDR")
	DEFAULT_SERVICE_CIDR      = spec.MustDefaultString("KlusterSpec", "serviceCIDR")
	DEFAULT_ADVERTISE_ADDRESS = spec.MustDefaultString("KlusterSpec", "advertiseAddress")
	DEFAULT_ADVERTISE_PORT    = spec.MustDefaultInt64("KlusterSpec", "advertisePort")
	DEFAULT_ETCD_BACKUP       = spec.MustDefaultString("KlusterSpec", "backup")
)

type KlusterFactory interface {
	KlusterFor(models.KlusterSpec) (*v1.Kluster, error)
}

type klusterFactory struct {
}

func NewKlusterFactory() KlusterFactory {
	return &klusterFactory{}
}

func (klusterFactory) KlusterFor(spec models.KlusterSpec) (*v1.Kluster, error) {
	if spec.Name == "" {
		return nil, fmt.Errorf("unabled to create cluster. missing name")
	}
	if spec.NodePools == nil {
		spec.NodePools = []models.NodePool{}
	}

	// Enable dex and dashboard for new clusters
	spec.Dex = true
	spec.Dashboard = true

	k := &v1.Kluster{
		Spec: spec,
		Status: models.KlusterStatus{
			Phase:     models.KlusterPhasePending,
			NodePools: []models.NodePoolInfo{},
		},
	}

	if k.Spec.ClusterCIDR == "" {
		k.Spec.ClusterCIDR = DEFAULT_CLUSTER_CIDR
	}

	if k.Spec.ServiceCIDR == "" {
		k.Spec.ServiceCIDR = DEFAULT_SERVICE_CIDR
	}

	if k.Spec.AdvertiseAddress == "" {
		k.Spec.AdvertiseAddress = DEFAULT_ADVERTISE_ADDRESS
	}
	if k.Spec.AdvertisePort == 0 {
		k.Spec.AdvertisePort = DEFAULT_ADVERTISE_PORT
	}
	_, serviceCIDR, err := net.ParseCIDR(k.Spec.ServiceCIDR)
	if err != nil {
		return nil, err
	}
	dnsip, err := ip.GetIndexedIP(serviceCIDR, 2)
	if err != nil {
		return nil, err
	}

	if k.Spec.DNSAddress == "" {
		k.Spec.DNSAddress = dnsip.String()
	}

	if k.Spec.DNSDomain == "" {
		k.Spec.DNSDomain = dns.DEFAULT_DOMAIN
	}

	if k.Spec.Backup == "" {
		k.Spec.Backup = DEFAULT_ETCD_BACKUP
	}

	if k.ObjectMeta.Name == "" {
		k.ObjectMeta.Name = spec.Name
	}

	if k.Status.Version == "" {
		k.Status.Version = version.GitCommit
	}

	for _, nodePool := range k.Spec.NodePools {
		k.Status.NodePools = append(k.Status.NodePools, models.NodePoolInfo{
			Name:        nodePool.Name,
			Size:        nodePool.Size,
			Running:     0,
			Healthy:     0,
			Schedulable: 0,
		})
	}

	k.Status.SpecVersion = int64(migration.Latest())

	return k, nil
}
