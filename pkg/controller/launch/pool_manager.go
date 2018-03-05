package launch

import (
	"fmt"
	"strings"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	"github.com/sapcc/kubernikus/pkg/templates"
	"github.com/sapcc/kubernikus/pkg/util"

	"github.com/go-kit/kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type PoolManager interface {
	GetStatus() (*PoolStatus, error)
	SetStatus(*PoolStatus) error
	CreateNode() (string, error)
	DeleteNode(string) error
}

type PoolStatus struct {
	Nodes       []string
	Running     int
	Starting    int
	Stopping    int
	Needed      int
	UnNeeded    int
	Healthy     int
	Schedulable int
}

type ConcretePoolManager struct {
	config.Clients
	nodeObservatory *nodeobservatory.NodeObservatory

	Kluster *v1.Kluster
	Pool    *models.NodePool
	Logger  log.Logger
}

func (lr *LaunchReconciler) newPoolManager(kluster *v1.Kluster, pool *models.NodePool) PoolManager {
	logger := log.With(lr.Logger,
		"kluster", kluster.Spec.Name,
		"project", kluster.Account(),
		"pool", pool.Name)

	var pm PoolManager
	pm = &ConcretePoolManager{lr.Clients, lr.nodeObervatory, kluster, pool, logger}
	pm = &EventingPoolManager{pm, kluster, lr.Recorder}
	pm = &LoggingPoolManager{pm, logger}
	pm = &InstrumentingPoolManager{pm,
		metrics.LaunchOperationsLatency,
		metrics.LaunchOperationsTotal,
		metrics.LaunchSuccessfulOperationsTotal,
		metrics.LaunchFailedOperationsTotal,
	}

	return pm
}

func (cpm *ConcretePoolManager) GetStatus() (status *PoolStatus, err error) {
	status = &PoolStatus{}
	nodes, err := cpm.Clients.Openstack.GetNodes(cpm.Kluster, cpm.Pool)
	if err != nil {
		return status, err
	}
	healthy, schedulable := cpm.healthyAndSchedulable()

	return &PoolStatus{
		Nodes:       cpm.nodeIDs(nodes),
		Running:     cpm.running(nodes),
		Starting:    cpm.starting(nodes),
		Stopping:    cpm.stopping(nodes),
		Needed:      cpm.needed(nodes),
		UnNeeded:    cpm.unNeeded(nodes),
		Healthy:     healthy,
		Schedulable: schedulable,
	}, nil
}

func nodePoolInfoGet(pool []models.NodePoolInfo, name string) (int, bool) {
	for i, node := range pool {
		if node.Name == name {
			return i, true
		}
	}
	return 0, false
}

func nodePoolSpecGet(pool []models.NodePool, name string) (int, bool) {
	for i, node := range pool {
		if node.Name == name {
			return i, true
		}
	}
	return 0, false
}

func removeNodePool(pool []models.NodePoolInfo, index int) []models.NodePoolInfo {
	return append(pool[:index], pool[index+1:]...)
}

func (cpm *ConcretePoolManager) SetStatus(status *PoolStatus) error {

	healthy, schedulable := cpm.healthyAndSchedulable()

	newInfo := models.NodePoolInfo{
		Name:        cpm.Pool.Name,
		Size:        cpm.Pool.Size,
		Running:     int64(status.Running + status.Starting),
		Healthy:     int64(healthy),
		Schedulable: int64(schedulable),
	}

	metrics.SetMetricNodePoolStatus(
		cpm.Kluster.GetName(),
		cpm.Pool.Name,
		map[string]int64{
			"running":     newInfo.Running,
			"healthy":     newInfo.Healthy,
			"schedulable": newInfo.Schedulable,
		},
	)

	//TODO: Use util.UpdateKlusterWithRetries here
	copy, err := cpm.Clients.Kubernikus.Kubernikus().Klusters(cpm.Kluster.Namespace).Get(cpm.Kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Add new pools in the spec to the status
	for _, specPool := range copy.Spec.NodePools {
		// Find the pool
		if npi, ok := nodePoolInfoGet(copy.Status.NodePools, specPool.Name); ok {
			// is there a need to update?
			if copy.Status.NodePools[npi] == newInfo {
				return nil
			}
			copy.Status.NodePools[npi] = newInfo
			_, err = cpm.Clients.Kubernikus.Kubernikus().Klusters(copy.Namespace).Update(copy)
			return err
			// not found so add it
		} else {
			copy.Status.NodePools = append(copy.Status.NodePools, newInfo)
			_, err = cpm.Clients.Kubernikus.Kubernikus().Klusters(copy.Namespace).Update(copy)
			return err
		}
	}

	// Delete pools from the status that are not in spec
	for i, infoPool := range copy.Status.NodePools {
		// find the pool
		if _, ok := nodePoolSpecGet(copy.Spec.NodePools, infoPool.Name); !ok {
			// not found in the spec anymore so delete it
			copy.Status.NodePools = removeNodePool(copy.Status.NodePools, i)
			_, err = cpm.Clients.Kubernikus.Kubernikus().Klusters(copy.Namespace).Update(copy)
			return err
		}
	}

	return nil
}

func (cpm *ConcretePoolManager) CreateNode() (id string, err error) {
	secret, err := cpm.Clients.Kubernetes.CoreV1().Secrets(cpm.Kluster.Namespace).Get(cpm.Kluster.GetName(), metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	userdata, err := templates.Ignition.GenerateNode(cpm.Kluster, secret, cpm.Logger)
	if err != nil {
		return "", err
	}

	id, err = cpm.Clients.Openstack.CreateNode(cpm.Kluster, cpm.Pool, userdata)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (cpm *ConcretePoolManager) DeleteNode(id string) (err error) {
	if err = cpm.Clients.Openstack.DeleteNode(cpm.Kluster, id); err != nil {
		return err
	}
	return nil
}

func (cpm *ConcretePoolManager) nodeIDs(nodes []openstack.Node) []string {
	result := []string{}
	for _, n := range nodes {
		result = append(result, n.ID)
	}
	return result
}

func (cpm *ConcretePoolManager) starting(nodes []openstack.Node) int {
	var count int = 0
	for _, n := range nodes {
		if n.Starting() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) stopping(nodes []openstack.Node) int {
	var count int = 0
	for _, n := range nodes {
		if n.Stopping() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) running(nodes []openstack.Node) int {
	var count int = 0
	for _, n := range nodes {
		if n.Running() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) needed(nodes []openstack.Node) int {
	needed := int(cpm.Pool.Size) - cpm.running(nodes) - cpm.starting(nodes)
	if needed < 0 {
		return 0
	}
	return needed
}

func (cpm ConcretePoolManager) unNeeded(nodes []openstack.Node) int {
	unneeded := cpm.running(nodes) + cpm.starting(nodes) - int(cpm.Pool.Size)
	if unneeded < 0 {
		return 0
	}
	return unneeded
}

func (cpm *ConcretePoolManager) healthyAndSchedulable() (healthy int, schedulable int) {
	nodeLister, err := cpm.nodeObservatory.GetListerForKluster(cpm.Kluster)
	if err != nil {
		return
	}
	nodes, err := nodeLister.List(labels.Everything())
	if err != nil {
		return
	}
	for _, node := range nodes {
		//Does the node belong to this pool?
		if strings.HasPrefix(node.Name, fmt.Sprintf("%s-%s", cpm.Kluster.Spec.Name, cpm.Pool.Name)) {
			if !node.Spec.Unschedulable {
				schedulable++
			}
			if util.IsNodeReady(node) {
				healthy++
			}
		}
	}
	return
}
