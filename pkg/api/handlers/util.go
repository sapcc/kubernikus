package handlers

import (
	"encoding/json"

	"github.com/go-openapi/swag"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	"fmt"
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
	if strings.Contains(name,accountId) {
		return name
	}
	return fmt.Sprintf("%s-%s",name,accountId)
}

func createPatch(old, new *tprv1.Kluster) (patchBytes []byte, patchType types.PatchType, err error) {

	oldData, err := json.Marshal(old)
	if err != nil {
		return nil, types.StrategicMergePatchType, err
	}

	newData, err := json.Marshal(new)
	if err != nil {
		return nil, types.StrategicMergePatchType, err
	}

	patchBytes, err = strategicpatch.CreateTwoWayMergePatch(oldData, newData, tprv1.Kluster{})
	if err != nil {
		return nil, types.StrategicMergePatchType, err
	}

	return patchBytes, types.StrategicMergePatchType, nil

}
