package certs

import (
	"fmt"
	"time"

	kitlog "github.com/go-kit/kit/log"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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
		return fmt.Errorf("Couldn't get kluster secret: %s", err)
	}

	certFactory := util.NewCertificateFactory(kluster, &secret.Certificates, cc.config.Kubernikus.Domain)
	err, updates := certFactory.Ensure()
	if err != nil {
		return fmt.Errorf("Certificate renewal failed: %s", err)
	}

	if len(updates) > 0 {
		err = util.UpdateKlusterSecret(cc.client, kluster, secret)
		if err != nil {
			return fmt.Errorf("Couldn't update kluster secret: %s", err)
		}

		listOpts := meta_v1.ListOptions{
			LabelSelector: fmt.Sprintf("app in (%s-etcd,%s-apiserver)", kluster.Name, kluster.Name),
			Limit:         2,
		}
		pods, err := cc.client.CoreV1().Pods(kluster.Namespace).List(listOpts)
		if err != nil {
			return fmt.Errorf("Couldn't list etcd/apiserver pods: %s", err)
		}

		deleteOpts := meta_v1.DeleteOptions{}
		for _, pod := range pods.Items {
			cc.logger.Log("msg", "Deleting pod", "pod", pod.Name)
			err = cc.client.CoreV1().Pods(kluster.Namespace).Delete(pod.Name, &deleteOpts)
			if err != nil {
				return fmt.Errorf("Couldn't delete pod %s: %s", pod.Name, err)
			}
		}

		cc.logger.Log("msg", "Certificates updated", "certificates", fmt.Sprintf("%v", updates))
	}

	return nil
}
