package certs

import (
	"context"
	"fmt"
	"time"

	kitlog "github.com/go-kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	"github.com/sapcc/kubernikus/pkg/util"
)

type certsController struct {
	logger     kitlog.Logger
	config     config.Config
	client     kubernetes.Interface
	kubernikus kubernikus_clientset.Interface
}

func New(syncPeriod time.Duration, factories config.Factories, config config.Config, clients config.Clients, logger kitlog.Logger) base.Controller {
	logger = kitlog.With(logger, "controller", "certs")

	certs := certsController{
		logger:     logger,
		config:     config,
		client:     clients.Kubernetes,
		kubernikus: clients.Kubernikus,
	}

	return base.NewPollingController(syncPeriod, factories.Kubernikus.Kubernikus().V1().Klusters(), &certs, logger)
}

func (cc *certsController) Reconcile(kluster *v1.Kluster) (err error) {
	secret, err := util.KlusterSecret(cc.client, kluster)
	if err != nil {
		return fmt.Errorf("couldn't get kluster secret: %s", err)
	}

	rotate := kluster.CARotation()

	certFactory := util.NewCertificateFactory(kluster, &secret.Certificates, cc.config.Kubernikus.Domain)
	updates, err := certFactory.Ensure(rotate)
	if err != nil {
		return fmt.Errorf("certificate renewal failed: %s", err)
	}

	if len(updates) > 0 {
		if err = util.UpdateKlusterSecret(cc.client, kluster, secret); err != nil {
			return fmt.Errorf("couldn't update kluster secret: %s", err)
		}
		cc.logger.Log("msg", "Certificates updated", "kluster", kluster.Name, "changes", fmt.Sprintf("%#v", updates))
	}

	if rotate {
		if err = cc.removeRotateCAAnnotation(kluster); err != nil {
			return fmt.Errorf("couldn't remove rotation annotation: %s", err)
		}
	}

	return nil
}

func (cc *certsController) removeRotateCAAnnotation(kluster *v1.Kluster) error {
	patch := fmt.Sprintf(`{"metadata":{"annotations":{%q:null}}}`, v1.RotateCAAnnotation)
	_, err := cc.kubernikus.KubernikusV1().Klusters(kluster.Namespace).Patch(
		context.TODO(),
		kluster.Name,
		types.MergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
	)
	return err
}
