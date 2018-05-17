package handlers

import (
	"fmt"

	"github.com/go-openapi/runtime/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cluster "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/generated/clientset/scheme"
	"github.com/sapcc/kubernikus/pkg/templates"
)

func NewGetClusterIgnition(rt *api.Runtime) operations.GetClusterIgnitionHandler {
	return &getClusterIgnition{rt}
}

type getClusterIgnition struct {
	*api.Runtime
}

func (d *getClusterIgnition) Handle(params operations.GetClusterIgnitionParams) middleware.Responder {
	kluster, err := d.Kubernikus.Kubernikus().Klusters(d.Namespace).Get(params.Name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "No Kluster found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	machines, err := d.ClusterAPI.Cluster().Machines(d.Namespace).List(metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	var found *cluster.Machine
	var config *v1.SAPCCloudProviderConfig
	for _, machine := range machines.Items {
		providerConfig, err := decodeConfig(machine)
		if err != nil {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
		}

		if providerConfig.Spec.IPXE == params.Mac {
			found = &machine
			config = providerConfig
			break
		}
	}

	if found == nil {
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "No machine with that MAC found")
	}

	secret, err := d.Kubernetes.CoreV1().Secrets(d.Namespace).Get(params.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 404, "Secret Not found")
		}
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	userdata, err := templates.Ignition.GenerateNode(kluster, found.Name, secret, found, config, d.Logger)
	if err != nil {
		return NewErrorResponse(&operations.GetClusterIgnitionDefault{}, 500, err.Error())
	}

	var ignition models.Ignition
	ignition = models.Ignition(string(userdata))

	return operations.NewGetClusterIgnitionOK().WithPayload(ignition)
}

func decodeConfig(machine cluster.Machine) (*v1.SAPCCloudProviderConfig, error) {
	obj, gvk, err := scheme.Codecs.UniversalDecoder(v1.SchemeGroupVersion).Decode(machine.Spec.ProviderConfig.Value.Raw, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decoding failure: %v", err)
	}
	config, ok := obj.(*v1.SAPCCloudProviderConfig)
	if !ok {
		return nil, fmt.Errorf("failure to cast to sapccloud; type: %v", gvk)
	}
	return config, nil
}
