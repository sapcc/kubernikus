package handlers

import (
	"github.com/go-openapi/swag"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"k8s.io/apimachinery/pkg/labels"
)

func modelsError(err error) *models.Error {
	return &models.Error{
		Message: swag.String(err.Error()),
	}
}

func accountSelector(principal *models.Principal) labels.Selector {
	return labels.SelectorFromSet(map[string]string{"account": principal.Account})
}
