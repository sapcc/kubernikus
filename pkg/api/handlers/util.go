package handlers

import (
	"github.com/go-openapi/swag"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
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

func clusterSpecNodePoolItemsFromTPR(k *v1.Kluster) []*models.ClusterSpecNodePoolsItems0 {
	items := make([]*models.ClusterSpecNodePoolsItems0, int64(len(k.Spec.NodePools)))
	for i, _ := range k.Spec.NodePools {
		items[i] = &models.ClusterSpecNodePoolsItems0{
			Name:   &k.Spec.NodePools[i].Name,
			Image:  k.Spec.NodePools[i].Image,
			Flavor: &k.Spec.NodePools[i].Flavor,
			Size:   &[]int64{int64(k.Spec.NodePools[i].Size)}[0],
		}
	}
	return items
}

func clusterStatusNodePoolItemsFromTPR(k *v1.Kluster) []*models.ClusterStatusNodePoolsItems0 {
	items := make([]*models.ClusterStatusNodePoolsItems0, int64(len(k.Status.NodePools)))
	for i, _ := range k.Status.NodePools {
		items[i] = &models.ClusterStatusNodePoolsItems0{
			Name:        &k.Status.NodePools[i].Name,
			Size:        &[]int64{int64(k.Status.NodePools[i].Size)}[0],
			Running:     &[]int64{int64(k.Status.NodePools[i].Running)}[0],
			Healthy:     &[]int64{int64(k.Status.NodePools[i].Healthy)}[0],
			Schedulable: &[]int64{int64(k.Status.NodePools[i].Schedulable)}[0],
		}
	}
	return items
}

func clusterModelFromTPR(k *v1.Kluster) *models.Cluster {
	return &models.Cluster{
		Name: swag.String(k.Spec.Name),
		Spec: &models.ClusterSpec{
			NodePools: clusterSpecNodePoolItemsFromTPR(k),
		},
		Status: &models.ClusterStatus{
			Kluster: &models.ClusterStatusKluster{
				State:   string(k.Status.Kluster.State),
				Message: k.Status.Kluster.Message,
			},
			NodePools: clusterStatusNodePoolItemsFromTPR(k),
		},
	}
}
