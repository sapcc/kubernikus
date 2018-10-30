package migration

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/admin"
	"github.com/sapcc/kubernikus/pkg/util"
)

func AddAggregationLayerCertificates(rawKluster []byte, kluster *v1.Kluster, client kubernetes.Interface, adminClient admin.AdminClient) (err error) {
	secret, err := client.CoreV1().Secrets(kluster.Namespace).Get(kluster.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	keys := []string{"aggregation-ca-key.pem", "aggregation-ca.pem", "aggregation-aggregator-key.pem", "aggregation-aggregator.pem"}
	missingCerts := false

	for _, key := range keys {
		if _, ok := secret.Data[key]; !ok {
			missingCerts = true
			break
		}
	}

	if !missingCerts {
		return nil
	}

	certs, err := util.CreateCertificates(kluster, "", "", "cluster.local")
	if err != nil {
		return err
	}

	for _, key := range keys {
		secret.Data[key] = []byte(certs[key])
	}

	_, err = client.CoreV1().Secrets(kluster.Namespace).Update(secret)

	return err
}
