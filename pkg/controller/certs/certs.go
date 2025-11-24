package certs

import (
	"fmt"
	"time"

	kitlog "github.com/go-kit/log"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

type certsController struct {
	logger kitlog.Logger
	config config.Config
	client kubernetes.Interface
}

func New(syncPeriod time.Duration, factories config.Factories, config config.Config, clients config.Clients, logger kitlog.Logger) base.Controller {
	logger = kitlog.With(logger, "controller", "certs")

	certs := certsController{
		logger: logger,
		config: config,
		client: clients.Kubernetes,
	}

	return base.NewPollingController(syncPeriod, factories.Kubernikus.Kubernikus().V1().Klusters(), &certs, logger)
}

func (cc *certsController) Reconcile(kluster *v1.Kluster) (err error) {
	secret, err := util.KlusterSecret(cc.client, kluster)
	if err != nil {
		return fmt.Errorf("couldn't get kluster secret: %s", err)
	}

	certFactory := util.NewCertificateFactory(kluster, &secret.Certificates, cc.config.Kubernikus.Domain)
	updates, err := certFactory.Ensure()
	if err != nil {
		return fmt.Errorf("certificate renewal failed: %s", err)
	}

	if len(updates) > 0 {
		err = util.UpdateKlusterSecret(cc.client, kluster, secret)
		if err != nil {
			return fmt.Errorf("couldn't update kluster secret: %s", err)
		}

		cc.logger.Log("msg", "Certificates updated", "kluster", kluster.Name, "changes", fmt.Sprintf("%#v", updates))
	}

	return nil
}
