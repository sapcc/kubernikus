package routegc

import (
	"net"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/pagination"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/version"
)

type RouteGarbageCollector struct {
	authOpts    tokens.AuthOptions
	compute     *gophercloud.ServiceClient
	network     *gophercloud.ServiceClient
	routerID    string
	clusterCIDR *net.IPNet
	logger      log.Logger
}

func New(authOpts tokens.AuthOptions, routerID string, clusterCIDR *net.IPNet) *RouteGarbageCollector {
	return &RouteGarbageCollector{
		authOpts:    authOpts,
		routerID:    routerID,
		clusterCIDR: clusterCIDR,
	}

}

func (r *RouteGarbageCollector) Run(logger log.Logger, syncPeriod time.Duration, stopCh <-chan struct{}) error {
	r.logger = log.With(logger, "controller", "routegc")
	r.logger.Log("msg", "Starting", "version", version.GitCommit, "interval", syncPeriod)

	defer r.logger.Log("msg", "Stopped")
	client, err := openstack.NewClient(r.authOpts.IdentityEndpoint)
	if err != nil {
		return err
	}
	if err := openstack.AuthenticateV3(client, &r.authOpts, gophercloud.EndpointOpts{}); err != nil {
		return err
	}
	if r.compute, err = openstack.NewComputeV2(client, gophercloud.EndpointOpts{}); err != nil {
		return err
	}
	if r.network, err = openstack.NewNetworkV2(client, gophercloud.EndpointOpts{}); err != nil {
		return err
	}
	wait.Until(r.Reconcile, syncPeriod, stopCh)
	return nil
}

func (r *RouteGarbageCollector) Reconcile() {
	defer func(begin time.Time) {
		r.logger.Log("msg", "Reconciling", "took", time.Since(begin), "v", 2)
	}(time.Now())

	validNexthops := map[string]string{}
	err := foreachServer(r.compute, servers.ListOpts{}, func(srv *servers.Server) (bool, error) {
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
		r.logger.Log("msg", "list servers", "err", err)
		return
	}

	logger := log.With(r.logger, "router", r.routerID)

	router, err := routers.Get(r.network, r.routerID).Extract()
	if err != nil {
		logger.Log("msg", "show router", "err", err)
		return
	}
	newRoutes := make([]routers.Route, 0, len(router.Routes))
	updated := false
	for _, route := range router.Routes {
		if r.isResponsibleForRoute(route) {
			if _, ok := validNexthops[route.NextHop]; !ok {
				updated = true
				logger.Log("msg", "route orphaned", "cidr", route.DestinationCIDR, "nexthop", route.NextHop)
				continue //delete the route (by not adding to newRoutes)
			}
		}
		newRoutes = append(newRoutes, route)
	}

	//something was changed, update the router
	if updated {
		_, err := routers.Update(r.network, r.routerID, routers.UpdateOpts{
			Routes: newRoutes,
		}).Extract()

		logger.Log("msg", "removed routes", "err", err)

	}

}

//adapted from  k8s.io/pkg/controller/route
func (r *RouteGarbageCollector) isResponsibleForRoute(route routers.Route) bool {
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {

		r.logger.Log("msg", "unparsable CIDR", "cidr", route.DestinationCIDR, "err", err)
		return false
	}
	// Not responsible if this route's CIDR is not within our clusterCIDR
	lastIP := make([]byte, len(cidr.IP))
	for i := range lastIP {
		lastIP[i] = cidr.IP[i] | ^cidr.Mask[i]
	}
	if !r.clusterCIDR.Contains(cidr.IP) || !r.clusterCIDR.Contains(lastIP) {
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
