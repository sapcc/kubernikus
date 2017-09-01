package handlers

import (
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"fmt"
	"strings"
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

func editCluster(clients *api.Clients, principal *models.Principal, name string, updateFunc func(k *v1.Kluster)) (*v1.Kluster, error) {
	kluster, err := clients.Kubernikus.Kubernikus().Klusters("kubernikus").Get(qualifiedName(name, principal.Account), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	updateFunc(kluster)

	updatedCluster, err := clients.Kubernikus.Kubernikus().Klusters("kubernikus").Update(kluster)
	if err != nil {
		return nil, err
	}
	return updatedCluster, nil

}
