package handlers

import (
	"fmt"
	"net/http"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
)

var (
	DEFAULT_IMAGE = spec.MustDefaultString("NodePool", "image")
)

func accountSelector(principal *models.Principal) labels.Selector {
	return labels.SelectorFromSet(map[string]string{"account": principal.Account})
}

// qualifiedName returns <cluster_name>-<account_id>
func qualifiedName(name string, accountId string) string {
	if strings.Contains(name, accountId) {
		return name
	}
	return fmt.Sprintf("%s-%s", name, accountId)
}

func editCluster(client kubernikusv1.KlusterInterface, principal *models.Principal, name string, updateFunc func(k *v1.Kluster)) (*v1.Kluster, error) {
	kluster, err := client.Get(qualifiedName(name, principal.Account), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	updateFunc(kluster)

	updatedCluster, err := client.Update(kluster)
	if err != nil {
		return nil, err
	}
	return updatedCluster, nil

}

func klusterFromCRD(k *v1.Kluster) *models.Kluster {
	return &models.Kluster{
		Name:   k.Spec.Name,
		Spec:   k.Spec,
		Status: k.Status,
	}
}

func getTracingLogger(request *http.Request) kitlog.Logger {
	logger, ok := request.Context().Value("logger").(kitlog.Logger)
	if !ok {
		logger = kitlog.NewNopLogger()
	}
	return logger
}
