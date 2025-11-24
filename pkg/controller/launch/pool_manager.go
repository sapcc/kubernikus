package launch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-kit/log"
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	kubernikus_listers "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/templates"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/bootstraptoken"
	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/pkg/version"
)

type PoolManager interface {
	GetStatus() (*PoolStatus, error)
	SetStatus(*PoolStatus) error
	CreateNode() (string, error)
	DeleteNode(string) error
	DeletePool() error
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
	Clients config.Clients

	klusterClient   openstack_kluster.KlusterClient
	nodeObservatory *nodeobservatory.NodeObservatory
	imageRegistry   version.ImageRegistry

	Kluster *v1.Kluster
	Pool    *models.NodePool
	Logger  log.Logger
	Lister  kubernikus_listers.KlusterLister
}

func (lr *LaunchReconciler) newPoolManager(kluster *v1.Kluster, pool *models.NodePool) (PoolManager, error) {
	logger := log.With(lr.Logger,
		"kluster", kluster.Spec.Name,
		"project", kluster.Account(),
		"pool", pool.Name)

	klusterClient, err := lr.Factories.Openstack.KlusterClientFor(kluster)
	if err != nil {
		return nil, err
	}

	var pm PoolManager
	pm = &ConcretePoolManager{lr.Clients, klusterClient, lr.nodeObervatory, lr.imageRegistry, kluster, pool, logger, lr.klusterInformer.Lister()}
	pm = &EventingPoolManager{pm, kluster, lr.Recorder}
	pm = &LoggingPoolManager{pm, logger}
	pm = &InstrumentingPoolManager{pm,
		metrics.LaunchOperationsLatency,
		metrics.LaunchOperationsTotal,
		metrics.LaunchSuccessfulOperationsTotal,
		metrics.LaunchFailedOperationsTotal,
	}

	return pm, nil
}

func (cpm *ConcretePoolManager) GetStatus() (status *PoolStatus, err error) {
	status = &PoolStatus{}

	nodes, err := cpm.klusterClient.ListNodes(cpm.Kluster, cpm.Pool)
	if err != nil {
		return status, err
	}
	healthy, schedulable := cpm.healthyAndSchedulable()

	nodesIDs := cpm.sortByUnschedulableNodes(cpm.nodeIDs(nodes))

	return &PoolStatus{
		Nodes:       nodesIDs,
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

func removeNodePool(pool []models.NodePoolInfo, name string) ([]models.NodePoolInfo, error) {
	index, ok := nodePoolInfoGet(pool, name)
	if !ok {
		return nil, fmt.Errorf("failed to delete PoolInfo: %s", name)
	}
	return append(pool[:index], pool[index+1:]...), nil
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

	copy, err := cpm.Clients.Kubernikus.KubernikusV1().Klusters(cpm.Kluster.Namespace).Get(context.TODO(), cpm.Kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	updated := false
	// Add new pools in the spec to the status
	// Find the pool
	if npi, ok := nodePoolInfoGet(copy.Status.NodePools, cpm.Pool.Name); ok {
		// is there a need to update?
		if copy.Status.NodePools[npi] != newInfo {
			copy.Status.NodePools[npi] = newInfo
			updated = true
		}
	} else {
		// not found so add it
		copy.Status.NodePools = append(copy.Status.NodePools, newInfo)
		updated = true
	}

	// Delete pools from the status that are not in spec
	// skip the pool if it is still in the spec
	if _, ok := nodePoolSpecGet(copy.Spec.NodePools, cpm.Pool.Name); !ok {
		// not found in the spec anymore so delete it
		copy.Status.NodePools, err = removeNodePool(copy.Status.NodePools, cpm.Pool.Name)
		if err != nil {
			return err
		}
		updated = true
	}

	if updated {
		_, err = util.UpdateKlusterWithRetries(cpm.Clients.Kubernikus.KubernikusV1().Klusters(cpm.Kluster.Namespace), cpm.Lister.Klusters(cpm.Kluster.Namespace), cpm.Kluster.GetName(), func(kluster *v1.Kluster) error {
			kluster.Status.NodePools = copy.Status.NodePools
			return nil
		})
	}
	return err
}

func (cpm *ConcretePoolManager) CreateNode() (id string, err error) {

	secret, err := util.KlusterSecret(cpm.Clients.Kubernetes, cpm.Kluster)
	if err != nil {
		return "", err
	}

	nodeName := generator.SimpleNameGenerator.GenerateName(fmt.Sprintf(util.NODE_NAMING_PATTERN_PREFIX, cpm.Kluster.Spec.Name, cpm.Pool.Name))

	client, err := cpm.Clients.Satellites.ClientFor(cpm.Kluster)
	if err != nil {
		return "", fmt.Errorf("couldn't get client for kluster: %s", err)
	}

	calicoNetworking := false
	if _, err := client.AppsV1().DaemonSets("kube-system").Get(context.TODO(), "calico-node", metav1.GetOptions{}); err == nil {
		calicoNetworking = true
	}

	token, tokenSecret, err := bootstraptoken.GenerateBootstrapToken(30 * time.Minute)
	if err != nil {
		return "", fmt.Errorf("node bootstrap token generation failed: %s", err)
	}

	if _, err := client.CoreV1().Secrets(tokenSecret.Namespace).Create(context.TODO(), tokenSecret, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("node bootstrap token secret creation failed: %s", err)
	}

	userdata, err := templates.Ignition.GenerateNode(cpm.Kluster, cpm.Pool, nodeName, token, secret, calicoNetworking, cpm.imageRegistry, cpm.Logger)
	if err != nil {
		return "", err
	}

	// add the AZ to this call
	id, err = cpm.klusterClient.CreateNode(cpm.Kluster, cpm.Pool, nodeName, userdata)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (cpm *ConcretePoolManager) DeleteNode(id string) (err error) {
	if err = cpm.klusterClient.DeleteNode(id); err != nil {
		return err
	}
	return nil
}

func (cpm *ConcretePoolManager) DeletePool() error {
	return cpm.klusterClient.DeleteServerGroup(cpm.Kluster.Name + "/" + cpm.Pool.Name)
}

func (cpm *ConcretePoolManager) nodeIDs(nodes []openstack_kluster.Node) []string {
	result := []string{}
	for _, n := range nodes {
		result = append(result, n.ID)
	}
	return result
}

func (cpm *ConcretePoolManager) starting(nodes []openstack_kluster.Node) int {
	var count int
	for _, n := range nodes {
		if n.Starting() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) stopping(nodes []openstack_kluster.Node) int {
	var count int
	for _, n := range nodes {
		if n.Stopping() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) running(nodes []openstack_kluster.Node) int {
	var count int
	for _, n := range nodes {
		if n.Running() {
			count = count + 1
		}
	}
	return count
}

func (cpm *ConcretePoolManager) needed(nodes []openstack_kluster.Node) int {
	needed := int(cpm.Pool.Size) - cpm.running(nodes) - cpm.starting(nodes)
	if needed < 0 {
		return 0
	}
	return needed
}

func (cpm ConcretePoolManager) unNeeded(nodes []openstack_kluster.Node) int {
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
		if util.IsKubernikusNode(node.Name, cpm.Kluster.Spec.Name, cpm.Pool.Name) {
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

func (cpm *ConcretePoolManager) sortByUnschedulableNodes(nodeIDs []string) []string {
	nodeLister, err := cpm.nodeObservatory.GetListerForKluster(cpm.Kluster)
	if err != nil {
		return nil
	}

	nodes, err := nodeLister.List(labels.Everything())
	if err != nil {
		return nil
	}

	kubernetesIDs := make(map[string]*core_v1.Node)
	for i := range nodes {
		id := strings.Replace(nodes[i].Spec.ProviderID, "openstack:///", "", 1)
		kubernetesIDs[id] = nodes[i]
	}

	sort.SliceStable(nodeIDs, func(i, j int) bool {
		//func has to return true only if i is actually higher priority

		iNode := kubernetesIDs[nodeIDs[i]]
		jNode := kubernetesIDs[nodeIDs[j]]

		//if i is not in K8S and j is --> i goes in front
		if iNode == nil && jNode != nil {
			return true
		}
		// if i is unschedulable and j is --> i goes in front
		if iNode != nil && iNode.Spec.Unschedulable && jNode != nil && !jNode.Spec.Unschedulable {
			return true
		}
		// in all other cases i does not have higher ranking
		return false
	})

	return nodeIDs
}
