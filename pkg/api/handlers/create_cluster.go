package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/validate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	k8sutil "github.com/sapcc/kubernikus/pkg/util/k8s"
)

func NewCreateCluster(rt *api.Runtime) operations.CreateClusterHandler {
	return &createCluster{rt}
}

type createCluster struct {
	*api.Runtime
}

func (d *createCluster) Handle(params operations.CreateClusterParams, principal *models.Principal) middleware.Responder {
	logger := getTracingLogger(params.HTTPRequest)
	name := params.Body.Name
	spec := params.Body.Spec

	if err := validate.UniqueItems("name", "body", params.Body.Spec.NodePools); err != nil {
		return NewErrorResponse(&operations.CreateClusterDefault{}, int(err.Code()), err.Error())
	}

	spec.Name = name
	for i, pool := range spec.NodePools {
		//Set default image
		if pool.Image == "" {
			spec.NodePools[i].Image = DEFAULT_IMAGE
		}
	}
	kluster, err := kubernikus.NewKlusterFactory().KlusterFor(spec)

	kluster.ObjectMeta = metav1.ObjectMeta{
		Name:        qualifiedName(name, principal.Account),
		Labels:      map[string]string{"account": principal.Account},
		Annotations: map[string]string{"creator": principal.Name},
	}

	k8sutil.EnsureNamespace(d.Kubernetes, d.Namespace)
	kluster, err = d.Kubernikus.Kubernikus().Klusters(d.Namespace).Create(kluster)
	if err != nil {
		logger.Log(
			"msg", "failed to create cluster",
			"kluster", kluster.GetName(),
			"project", kluster.Account(),
			"err", err)

		if apierrors.IsAlreadyExists(err) {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Cluster with name %s already exists", name)
		}
		return NewErrorResponse(&operations.CreateClusterDefault{}, 500, err.Error())
	}

	return operations.NewCreateClusterCreated().WithPayload(klusterFromCRD(kluster))
}
