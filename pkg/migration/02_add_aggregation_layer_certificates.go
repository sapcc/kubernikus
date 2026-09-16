package migration

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

func AddAggregationLayerCertificates(rawKluster []byte, kluster *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	apiSecret, err := clients.Kubernetes.CoreV1().Secrets(kluster.Namespace).Get(context.TODO(), kluster.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	secret, err := v1.NewSecret(apiSecret)
	if err != nil {
		return err
	}

	if secret.AggregationAggregatorCertificate != "" && secret.AggregationCACertificate != "" && secret.AggregationAggregatorPrivateKey != "" && secret.AggregationCAPrivateKey != "" {
		return nil
	}

	factory := util.NewCertificateFactory(kluster, &secret.Certificates, "")

	if _, err := factory.Ensure(); err != nil {
		return err
	}

	secretData, err := secret.ToData()
	if err != nil {
		return err
	}

	//only copy the aggregetion part because factory.Ensure() regenerates all non CA certs
	for _, key := range []string{"aggregation-ca-key.pem", "aggregation-ca.pem", "aggregation-aggregator-key.pem", "aggregation-aggregator.pem"} {
		apiSecret.Data[key] = secretData[key]
	}
	_, err = clients.Kubernetes.CoreV1().Secrets(kluster.Namespace).Update(context.TODO(), apiSecret, metav1.UpdateOptions{})

	return err
}
