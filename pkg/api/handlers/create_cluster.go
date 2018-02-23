package handlers

import (
	"net"

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
	return &createCluster{Runtime: rt}
}

type createCluster struct {
	*api.Runtime
	cpServiceIP net.IP
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
	if err != nil {
		logger.Log(
			"msg", "failed to create cluster",
			"kluster", name,
			"project", principal.Account,
			"err", err)
		return NewErrorResponse(&operations.CreateClusterDefault{}, 400, err.Error())
	}

	//Ensure that the service CIDR range does not overlap with the control plane service CIDR
	//Otherwise the wormhole server will prevent the kluster apiserver from reaching its etcd
	if _, svcCIDR, err := net.ParseCIDR(kluster.Spec.ServiceCIDR); err == nil {
		if svcIP := d.controlPlaneServiceIP(); svcIP != nil && svcCIDR.Contains(svcIP) {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Service CIDR %s not allowed", kluster.Spec.ServiceCIDR)
		}
	}

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

//get (and cache) the kubernetes apiserver service ip of the control plane
func (d *createCluster) controlPlaneServiceIP() net.IP {
	if d.cpServiceIP != nil {
		return d.cpServiceIP
	}
	svc, err := d.Kubernetes.Core().Services("default").Get("kubernetes", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	d.cpServiceIP = net.ParseIP(svc.Spec.ClusterIP)
	return d.cpServiceIP
}
