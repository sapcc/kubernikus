package routegc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/version"
)

type RouteGarbageCollector struct {
	config.Factories
	logger       log.Logger
	watchers     sync.Map
	syncPeriod   time.Duration
	klusterIndex cache.Indexer
}

func New(syncPeriod time.Duration, factories config.Factories, logger log.Logger) *RouteGarbageCollector {
	gc := &RouteGarbageCollector{
		Factories:    factories,
		logger:       log.With(logger, "controller", "routegc"),
		syncPeriod:   syncPeriod,
		klusterIndex: factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer(),
	}

	factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    gc.klusterAdd,
			DeleteFunc: gc.klusterDelete,
		})
	return gc

}

func (r *RouteGarbageCollector) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	r.logger.Log("msg", "Starting", "version", version.GitCommit, "interval", r.syncPeriod)
	defer r.logger.Log("msg", "Stopped")
	<-stopCh
	//Stop all reconciliation loops
	r.watchers.Range(func(key, value interface{}) bool {
		close(value.(chan struct{}))
		return true
	})
}

func (r *RouteGarbageCollector) reconcile(kluster *v1.Kluster, logger log.Logger) error {
	routerID := kluster.Spec.Openstack.RouterID
	defer func(begin time.Time) {
	}(time.Now())

	providerClient, err := r.Openstack.ProviderClientForKluster(kluster, logger)
	if err != nil {
		return err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Failed to setup openstack network client: %s", err)
	}

	_, clusterCIDR, err := net.ParseCIDR(kluster.Spec.ClusterCIDR)
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

	logger = log.With(logger, "router", routerID)

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
			Routes: newRoutes,
		}).Extract()
		if err != nil {
			return fmt.Errorf("Failed to remove routes: %s", err)
		}
		metrics.OrphanedRoutesTotal.With(prometheus.Labels{}).Add(float64(len(router.Routes) - len(newRoutes)))
		logger.Log("msg", "removed routes")
	}
	return nil

}

func (r *RouteGarbageCollector) klusterAdd(obj interface{}) {
	//TODO: Don't start routegc watchloop for 1.11+ klusters
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	closeCh := make(chan struct{})
	if _, alreadyStored := r.watchers.LoadOrStore(key, closeCh); alreadyStored {
		return
	}
	go r.watchKluster(key, closeCh)
}

func (r *RouteGarbageCollector) watchKluster(key string, stop <-chan struct{}) {
	reconcile := func() {
		obj, exists, err := r.klusterIndex.GetByKey(key)
		if !exists || err != nil {
			return
		}
		kluster := obj.(*v1.Kluster)
		logger := log.With(r.logger, "kluster", kluster.Name)
		begin := time.Now()
		err = r.reconcile(kluster, logger)
		logger.Log("msg", "Reconciling", "took", time.Since(begin), "v", 5, "err", err)
		if err != nil {
			metrics.RouteGCFailedOperationsTotal.With(prometheus.Labels{}).Add(1)
		}
	}
	wait.JitterUntil(reconcile, r.syncPeriod, 0.5, true, stop)
}

func (r *RouteGarbageCollector) klusterDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	if stopCh, found := r.watchers.Load(key); found {
		close(stopCh.(chan struct{}))
		r.watchers.Delete(key)
	}
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
