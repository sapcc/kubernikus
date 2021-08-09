package routegc

import (
	"fmt"
	"net"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	os_client "github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
)

type routeGarbageCollector struct {
	logger          log.Logger
	osClientFactory os_client.SharedOpenstackClientFactory
}

func New(syncPeriod time.Duration, factories config.Factories, logger log.Logger) base.Controller {

	logger = log.With(logger, "controller", "routegc")

	routeGC := routeGarbageCollector{
		logger:          logger,
		osClientFactory: factories.Openstack,
	}

	return base.NewPollingController(syncPeriod, factories.Kubernikus.Kubernikus().V1().Klusters(), &routeGC, logger)
}

func (w *routeGarbageCollector) Reconcile(kluster *v1.Kluster) (err error) {

	//skip klusters not in state Running
	if kluster.Status.Phase != models.KlusterPhaseRunning {
		return nil
	}

	// disable routegc for clusters with no cloudprovider
	if kluster.Spec.NoCloud {
		return nil
	}

	routerID := kluster.Spec.Openstack.RouterID
	defer func(begin time.Time) {
		if err != nil {
			metrics.RouteGCFailedOperationsTotal.With(prometheus.Labels{}).Add(1)
		}
	}(time.Now())

	providerClient, err := w.osClientFactory.ProviderClientForKluster(kluster, w.logger)
	if err != nil {
		return err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Failed to setup openstack network client: %s", err)
	}

	_, clusterCIDR, err := net.ParseCIDR(kluster.ClusterCIDR())
	if err != nil {
		return fmt.Errorf("Failed to parse clusterCIDR: %s", err)
	}

	router, err := routers.Get(networkClient, routerID).Extract()
	if err != nil {
		return fmt.Errorf("Failed to get router %s: %s", routerID, err)
	}

	if len(router.Routes) == 0 {
		return nil
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Failed to setup openstack compute client: %s", err)
	}

	validNexthops := map[string]string{}
	err = foreachServer(computeClient, servers.ListOpts{}, func(srv *servers.Server) (bool, error) {
		for _, addrs := range srv.Addresses {
			for _, nase := range addrs.([]interface{}) {
				addresses, ok := nase.(map[string]interface{})
				if !ok {
					continue
				}
				addr, ok := addresses["addr"]
				if !ok {
					continue
				}
				validNexthops[addr.(string)] = srv.ID
			}
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("Failed to list servers: %s", err)
	}

	logger := log.With(w.logger, "router", routerID)

	newRoutes := make([]routers.Route, 0, len(router.Routes))
	for _, route := range router.Routes {
		if isResponsibleForRoute(clusterCIDR, route) {
			if _, ok := validNexthops[route.NextHop]; !ok {
				logger.Log("msg", "route orphaned", "cidr", route.DestinationCIDR, "nexthop", route.NextHop)
				continue //delete the route (by not adding to newRoutes)
			}
		}
		newRoutes = append(newRoutes, route)
	}

	//something was changed, update the router
	if len(newRoutes) < len(router.Routes) {
		_, err := routers.Update(networkClient, routerID, routers.UpdateOpts{
			Routes: &newRoutes,
		}).Extract()
		if err != nil {
			return fmt.Errorf("Failed to remove routes: %s", err)
		}
		metrics.OrphanedRoutesTotal.With(prometheus.Labels{}).Add(float64(len(router.Routes) - len(newRoutes)))
		logger.Log("msg", "removed routes")
	}
	return nil

}

//adapted from  k8s.io/pkg/controller/route
func isResponsibleForRoute(clusterCIDR *net.IPNet, route routers.Route) bool {
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return false
	}
	// Not responsible if this route's CIDR is not within our clusterCIDR
	lastIP := make([]byte, len(cidr.IP))
	for i := range lastIP {
		lastIP[i] = cidr.IP[i] | ^cidr.Mask[i]
	}
	if !clusterCIDR.Contains(cidr.IP) || !clusterCIDR.Contains(lastIP) {
		return false
	}
	return true
}

//taken from k8s.io/pkg/cloudprovider/openstack/
func foreachServer(client *gophercloud.ServiceClient, opts servers.ListOptsBuilder, handler func(*servers.Server) (bool, error)) error {
	pager := servers.List(client, opts)

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		s, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		for _, server := range s {
			ok, err := handler(&server)
			if !ok || err != nil {
				return false, err
			}
		}
		return true, nil
	})
	return err
}
