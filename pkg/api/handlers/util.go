package handlers

import (
	"errors"
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

func editCluster(client kubernikusv1.KlusterInterface, principal *models.Principal, name string, updateFunc func(k *v1.Kluster) error) (*v1.Kluster, error) {
	kluster, err := client.Get(qualifiedName(name, principal.Account), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = updateFunc(kluster)
	if err != nil {
		return nil, err
	}

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

// detectNodePoolChanges checks for the changes between node pool lists
func detectNodePoolChanges(oldNodePools, newNodePools []models.NodePool) (nodePoolsToDelete []string, err error) {

	nodePoolsToDelete = make([]string, 0)

	// For each old node pool
	for _, old := range oldNodePools {
		foundInNew := false
		// For each new node pool
		for _, new := range newNodePools {
			// Found in both!
			if old.Name == new.Name {
				foundInNew = true

				err := nodePoolEqualsWithScaling(old, new)
				if err != nil {
					return nodePoolsToDelete, err
				}
			}
		}
		if !foundInNew {
			if old.Size != 0 {
				return nodePoolsToDelete, errors.New("nodepool with size larger than 0 cannot be deleted: " + old.Name)
			} else {
				nodePoolsToDelete = append(nodePoolsToDelete, old.Name)
			}

		}
	}

	return nodePoolsToDelete, nil
}

// nodePoolEqualsWithScaling checks whether the node pool is only scaled without any changes
func nodePoolEqualsWithScaling(old, new models.NodePool) error {

	if old.Flavor != new.Flavor || old.Image != new.Image || old.Name != new.Name {
		return errors.New("nodepool data cannot be changed except size: " + old.Name)
	}

	return nil
}
