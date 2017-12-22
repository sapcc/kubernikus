package kubernikus

import (
	"fmt"
	"net"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/ip"
	"github.com/sapcc/kubernikus/pkg/version"
)

var (
	//Keep this in sync with the default in swagger.yaml
	DEFAULT_CLUSTER_CIDR      = spec.MustDefaultString("KlusterSpec", "clusterCIDR")
	DEFAULT_SERVICE_CIDR      = spec.MustDefaultString("KlusterSpec", "serviceCIDR")
	DEFAULT_ADVERTISE_ADDRESS = spec.MustDefaultString("KlusterSpec", "advertiseAddress")
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

	if k.Spec.Version == "" {
		k.Spec.Version = util.DEFAULT_KUBERNETES_VERSION
	}

	if k.Spec.Version == "" {
		k.Spec.Version = util.DEFAULT_KUBERNETES_VERSION
	}

	if k.Spec.Version != util.DEFAULT_KUBERNETES_VERSION {
		return nil, fmt.Errorf("Unable to create cluster. Unsupported Kubernetes version.")
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

	return k, nil
}
