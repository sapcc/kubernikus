package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	kitlog "github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/spec"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	kubernikusv1 "github.com/sapcc/kubernikus/pkg/generated/clientset/typed/kubernikus/v1"
)

var (
	DEFAULT_IMAGE              = spec.MustDefaultString("NodePool", "image")
	FetchOpenstackMetadataFunc = fetchOpenstackMetadata
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
	kluster, err := client.Get(context.TODO(), qualifiedName(name, principal.Account), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = updateFunc(kluster)
	if err != nil {
		return nil, err
	}

	updatedCluster, err := client.Update(context.TODO(), kluster, metav1.UpdateOptions{})
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

func fetchOpenstackMetadata(request *http.Request, principal *models.Principal) (*models.OpenstackMetadata, error) {
	tokenID := request.Header.Get("X-Auth-Token")

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: auth.OpenStackAuthURL(),
		TokenID:          tokenID,
		Scope: tokens.Scope{
			ProjectID: principal.Account,
		},
	}

	client, err := openstack.NewSharedOpenstackClientFactory(nil, nil, nil, getTracingLogger(request)).ProjectClientFor(authOptions)
	if err != nil {
		return nil, err
	}

	return client.GetMetadata()
}
