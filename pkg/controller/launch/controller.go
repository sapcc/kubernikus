package launch

import (
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/templates"

	"github.com/go-kit/kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

type PoolManager interface {
	GetStatus() (*PoolStatus, error)
	SetStatus(*PoolStatus) error
	CreateNode() (string, error)
	DeleteNode(string) error
}

type LaunchReconciler struct {
	config.Clients

	Recorder record.EventRecorder
	Logger   log.Logger
}

type PoolStatus struct {
	Nodes    []string
	Running  int
	Starting int
	Stopping int
	Needed   int
	UnNeeded int
}

type ConcretePoolManager struct {
	config.Clients

	Kluster *v1.Kluster
	Pool    *models.NodePool
	Logger  log.Logger
}

func NewController(factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger,
		"controller", "launch")

	var reconciler base.Reconciler
	reconciler = &LaunchReconciler{clients, recorder, logger}
	reconciler = &base.LoggingReconciler{reconciler, logger}
	reconciler = &base.EventingReconciler{reconciler}
	reconciler = &base.InstrumentingReconciler{
		reconciler,
		metrics.LaunchOperationsLatency,
		metrics.LaunchOperationsTotal,
		metrics.LaunchSuccessfulOperationsTotal,
		metrics.LaunchFailedOperationsTotal,
	}

	return base.NewController(factories, clients, reconciler, logger)
}

func (lr *LaunchReconciler) Reconcile(kluster *v1.Kluster) (requeueRequested bool, err error) {
	if !(kluster.Status.Phase == models.KlusterPhaseRunning || kluster.Status.Phase == models.KlusterPhaseTerminating) {
		return false, nil
	}

	for _, pool := range kluster.Spec.NodePools {
		_, requeue, err := lr.reconcilePool(kluster, &pool)
		if err != nil {
			return false, err
		}

		if requeue {
			requeueRequested = true
		}
	}

	return requeueRequested, nil
}

func (lr *LaunchReconciler) reconcilePool(kluster *v1.Kluster, pool *models.NodePool) (status *PoolStatus, requeue bool, err error) {

	pm := lr.newNodePoolManager(kluster, pool)
	status, err = pm.GetStatus()
	if err != nil {
		return
	}

	switch {
	case kluster.Status.Phase == models.KlusterPhaseTerminating:
		for _, node := range status.Nodes {
			requeue = true
			if err = pm.DeleteNode(node); err != nil {
				return
			}
		}
		return
	case status.Needed > 0:
		for i := 0; i < int(status.Needed); i++ {
			requeue = true
			if _, err = pm.CreateNode(); err != nil {
				return
			}
		}
		return
	case status.UnNeeded > 0:
		for i := 0; i < int(status.UnNeeded); i++ {
			requeue = true
			if err = pm.DeleteNode(status.Nodes[i]); err != nil {
				return
			}
		}
		return
	case status.Starting > 0:
		requeue = true
	case status.Stopping > 0:
		requeue = true
	default:
		return
	}

	err = pm.SetStatus(status)
	return
}

func (lr *LaunchReconciler) newNodePoolManager(kluster *v1.Kluster, pool *models.NodePool) PoolManager {
	logger := log.With(lr.Logger,
		"kluster", kluster.Spec.Name,
		"project", kluster.Account(),
		"pool", pool.Name)

	var pm PoolManager
	pm = &ConcretePoolManager{lr.Clients, kluster, pool, logger}
	pm = &EventingPoolManager{pm, kluster, lr.Recorder}
	pm = &LoggingPoolManager{pm, logger}

	return pm
}

func (cpm *ConcretePoolManager) GetStatus() (status *PoolStatus, err error) {
	status = &PoolStatus{}
	nodes, err := cpm.Clients.Openstack.GetNodes(cpm.Kluster, cpm.Pool)
	if err != nil {
		return status, err
	}

	return &PoolStatus{
		Nodes:    cpm.nodeIDs(nodes),
		Running:  cpm.running(nodes),
		Starting: cpm.starting(nodes),
		Stopping: cpm.stopping(nodes),
		Needed:   cpm.needed(nodes),
		UnNeeded: cpm.unNeeded(nodes),
	}, nil
}

func (cpm *ConcretePoolManager) SetStatus(status *PoolStatus) error {
	newInfo := models.NodePoolInfo{
		Name:        cpm.Pool.Name,
		Size:        cpm.Pool.Size,
		Running:     int64(status.Running + status.Starting),
		Healthy:     int64(status.Running),
		Schedulable: int64(status.Running),
	}

	copy, err := cpm.Clients.Kubernikus.Kubernikus().Klusters(cpm.Kluster.Namespace).Get(cpm.Kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for i, curInfo := range copy.Status.NodePools {
		if curInfo.Name == newInfo.Name {
			if curInfo == newInfo {
				return nil
			}

			copy.Status.NodePools[i] = newInfo
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

	userdata, err := templates.Ignition.GenerateNode(cpm.Kluster, secret)
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
