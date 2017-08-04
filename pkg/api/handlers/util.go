package handlers

import (
	"github.com/go-openapi/swag"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"

	"fmt"
	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	"strings"
)

func modelsError(err error) *models.Error {
	return &models.Error{
		Message: swag.String(err.Error()),
	}
}

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

func editCluster(tprClient *rest.RESTClient, principal *models.Principal, name string, updateFunc func(k *tprv1.Kluster)) (*tprv1.Kluster, error) {
	var kluster, updatedCluster tprv1.Kluster
	if err := tprClient.Get().Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(qualifiedName(name, principal.Account)).Do().Into(&kluster); err != nil {
		return nil, err
	}

	updateFunc(&kluster)

	if err := tprClient.Put().Body(&kluster).Namespace("kubernikus").Resource(tprv1.KlusterResourcePlural).LabelsSelectorParam(accountSelector(principal)).Name(qualifiedName(name, principal.Account)).Do().Into(&updatedCluster); err != nil {
		return nil, err
	}
	return &updatedCluster, nil

}
