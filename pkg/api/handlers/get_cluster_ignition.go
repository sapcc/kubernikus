package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/templates"
)

func NewGetClusterIgnition(rt *api.Runtime) operations.GetClusterIgnitionHandler {
	return &getClusterIgnition{rt}
}

type getClusterIgnition struct {
	*api.Runtime
}

func (d *getClusterIgnition) Handle(params operations.GetClusterIgnitionParams, principal *models.Principal) middleware.Responder {
	name := qualifiedName(params.Name, principal.Account)
	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	secret, err := d.Kubernetes.CoreV1().Secrets(d.Namespace).Get(qualifiedName(params.Name, principal.Account), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	userdata, err := templates.Ignition.GenerateNode(kluster, secret, d.Logger)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	var ignition models.Ignition
	ignition = models.Ignition(string(userdata))

	return operations.NewGetClusterIgnitionOK().WithPayload(ignition)
}
