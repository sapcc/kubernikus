package hammertime

import (
	"fmt"
	"time"

	kitlog "github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	"github.com/sapcc/kubernikus/pkg/util"
)

type hammertimeController struct {
	nodeObervatory *nodeobservatory.NodeObservatory
	client         kubernetes.Interface
	timeout        time.Duration
	logger         kitlog.Logger
	recorder       record.EventRecorder
}

const (
	HammertimeDisableAnnotation = "kubernikus.cloud.sap/hammertime"
)

func New(syncPeriod time.Duration, timeout time.Duration, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger kitlog.Logger) base.Controller {

	logger = kitlog.With(logger, "controller", "hammertime")

	controller := hammertimeController{
		nodeObervatory: factories.NodesObservatory.NodeInformer(),
		client:         clients.Kubernetes,
		timeout:        timeout,
		logger:         logger,
		recorder:       recorder,
	}

	return base.NewPollingController(syncPeriod, factories.Kubernikus.Kubernikus().V1().Klusters(), &controller, logger)
}

func (hc *hammertimeController) Reconcile(kluster *v1.Kluster) error {
	logger := kitlog.With(hc.logger, "kluster", kluster.GetName())

	//Hammertime only  makes sense after the kluster's deployment exist (Duh we want to scale them)
	if kluster.Status.Phase == models.KlusterPhasePending || kluster.Status.Phase == models.KlusterPhaseCreating {
		return nil
	}

	// stop hammertime during upgrades and termination or if explicitly disabled
	if kluster.Status.Phase != models.KlusterPhaseRunning || util.DisabledValue(kluster.Annotations[HammertimeDisableAnnotation]) {
		return hc.scaleDeployment(kluster, false, logger)
	}

	lister, err := hc.nodeObervatory.GetListerForKluster(kluster)
	if err != nil {
		return fmt.Errorf("Failed to get node lister: %s", err)
	}
	nodes, err := lister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("listing nodes failed: %s", err)
	}

	// No Hammertime if the cluster is terminating or has less then two nodes
	specNodes := 0
	for _, pool := range kluster.Spec.NodePools {
		specNodes += int(pool.Size)
	}
	if len(nodes) < 2 || specNodes < 2 || kluster.Status.Phase != models.KlusterPhaseRunning {
		metrics.HammertimeStatus.WithLabelValues(kluster.Name).Set(0)
		return hc.scaleDeployment(kluster, false, logger)
	}

	//var oldestHeartbeat time.Time = time.Now()
	var newestHearbeat time.Time = time.Time{}
	for _, node := range nodes {
		ready := nodeReadyCondition(node)
		if ready == nil {
			continue
		}
		if ready.LastHeartbeatTime.After(newestHearbeat) {
			newestHearbeat = ready.LastHeartbeatTime.Time
		}
	}

	timeout_exeeded := time.Now().Sub(newestHearbeat) > hc.timeout
	if timeout_exeeded {
		metrics.HammertimeStatus.WithLabelValues(kluster.Name).Set(1)
	} else {
		metrics.HammertimeStatus.WithLabelValues(kluster.Name).Set(0)
	}

	return hc.scaleDeployment(kluster, timeout_exeeded, logger)

}

func (hc *hammertimeController) scaleDeployment(kluster *v1.Kluster, disable bool, logger kitlog.Logger) error {

	deploymentClient := hc.client.ExtensionsV1beta1().Deployments(kluster.Namespace)

	deploymentName := fmt.Sprintf("%s-cmanager", kluster.GetName())
	scale, err := deploymentClient.GetScale(deploymentName, meta_v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		deploymentName = fmt.Sprintf("%s-controller-manager", kluster.GetName())
		scale, err = deploymentClient.GetScale(deploymentName, meta_v1.GetOptions{})
	}
	if err != nil {
		return fmt.Errorf("Failed to get deployment scale: %s", err)
	}

	if scale.Spec.Replicas > 0 {
		if disable {
			scale.Spec.Replicas = 0
			_, err = deploymentClient.UpdateScale(deploymentName, scale)
			logger.Log("msg", "Scaling down", "deployment", deploymentName, "err", err)
		}
	} else {
		if !disable {
			scale.Spec.Replicas = 1
			_, err = deploymentClient.UpdateScale(deploymentName, scale)
			logger.Log("msg", "Scaling up", "deployment", deploymentName, "err", err)
		}
	}
	return err
}

func nodeReadyCondition(node *core_v1.Node) *core_v1.NodeCondition {
	for _, c := range node.Status.Conditions {
		if c.Type == core_v1.NodeReady {
			return &c
		}
	}
	return nil
}
