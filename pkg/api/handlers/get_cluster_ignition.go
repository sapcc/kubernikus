package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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

	nodes, err := d.Kubernikus.Kubernikus().ExternalNodes(d.Namespace).List(metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	var found *v1.ExternalNode
	for _, node := range nodes.Items {
		if node.Spec.IPXE == params.Mac {
			found = &node
			break
		}
	}

	if found == nil {
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
	}

	secret, err := d.Kubernetes.CoreV1().Secrets(d.Namespace).Get(qualifiedName(params.Name, principal.Account), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	userdata, err := templates.Ignition.GenerateNode(kluster, found.Name, secret, found, d.Logger)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	var ignition models.Ignition
	ignition = models.Ignition(string(userdata))

	return operations.NewGetClusterIgnitionOK().WithPayload(ignition)
}
