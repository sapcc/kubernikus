package handlers

import (
	"errors"
	"net"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/validate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus"
	"github.com/sapcc/kubernikus/pkg/util/ip"
	k8sutil "github.com/sapcc/kubernikus/pkg/util/k8s"
)

func NewCreateCluster(rt *api.Runtime) operations.CreateClusterHandler {
	return &createCluster{Runtime: rt}
}

type createCluster struct {
	*api.Runtime
	cpServiceCIDR *net.IPNet
	cpClusterCIDR *net.IPNet
}

func (d *createCluster) Handle(params operations.CreateClusterParams, principal *models.Principal) middleware.Responder {
	logger := getTracingLogger(params.HTTPRequest)
	name := params.Body.Name
	spec := params.Body.Spec

	if err := validate.UniqueItems("name", "body", params.Body.Spec.NodePools); err != nil {
		return NewErrorResponse(&operations.CreateClusterDefault{}, int(err.Code()), err.Error())
	}

	var metadata *models.OpenstackMetadata
	var defaultAVZ string
	if len(spec.NodePools) > 0 {
		m, err := FetchOpenstackMetadataFunc(params.HTTPRequest, principal)
		if err != nil {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 500, err.Error())
		}
		metadata = m

		avz, err := getDefaultAvailabilityZone(metadata)
		if err != nil {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 500, err.Error())
		}
		defaultAVZ = avz
	}

	spec.Name = name
	for i, pool := range spec.NodePools {
		// Set default image
		if pool.Image == "" {
			spec.NodePools[i].Image = DEFAULT_IMAGE
		}

		// Set default AvailabilityZone
		if pool.AvailabilityZone == "" {
			spec.NodePools[i].AvailabilityZone = defaultAVZ
		}

		// Validate AVZ
		if err := validateAavailabilityZone(spec.NodePools[i].AvailabilityZone, metadata); err != nil {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Availability Zone %s is invalid: %s", spec.NodePools[i].AvailabilityZone, err)
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

	//Ensure that the service CIDR range does not overlap with any control plane CIDR
	//Otherwise the wormhole server will prevent the kluster apiserver from functioning properly
	if overlap, err := d.overlapWithControlPlane(kluster.Spec.ServiceCIDR); overlap {
		return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Service CIDR %s not allowed: %s", kluster.Spec.ServiceCIDR, err)
	}
	//Ensure that the cluster CIDR range does not overlap with any control plane CIDR
	//Otherwise the wormhole server will prevent the kluster apiserver from functioning properly
	if overlap, err := d.overlapWithControlPlane(kluster.Spec.ClusterCIDR); overlap {
		return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Cluster CIDR %s not allowed: %s", kluster.Spec.ClusterCIDR, err)
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

func (d *createCluster) overlapWithControlPlane(cidr string) (bool, error) {
	_, inputCIDR, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	clusterCIDR := d.controlPlaneClusterCIDR()
	if clusterCIDR != nil && ip.CIDROverlap(inputCIDR, clusterCIDR) {
		return true, errors.New("overlap with control plane cluster CIDR")
	}
	svcCIDR := d.controlPlaneServiceCIDR()
	if svcCIDR != nil && ip.CIDROverlap(inputCIDR, svcCIDR) {
		return true, errors.New("overlap with control plane service CIDR")
	}
	return false, nil
}

//approximate the control plane service CIDR by getting one service IP and assuming a /17 prefix
func (d *createCluster) controlPlaneServiceCIDR() *net.IPNet {
	if d.cpServiceCIDR != nil {
		return d.cpServiceCIDR
	}
	svc, err := d.Kubernetes.Core().Services("default").Get("kubernetes", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	_, ipnet, err := net.ParseCIDR(svc.Spec.ClusterIP + "/17")
	if err != nil {
		return nil
	}
	d.cpServiceCIDR = ipnet
	return d.cpServiceCIDR
}

//we infer the clusterCIDR by taking a Pod IP and assuming /16
func (d *createCluster) controlPlaneClusterCIDR() *net.IPNet {
	if d.cpClusterCIDR != nil {
		return d.cpClusterCIDR
	}
	podList, err := d.Kubernetes.Core().Pods(metav1.NamespaceAll).List(metav1.ListOptions{Limit: 1})
	if err != nil || len(podList.Items) == 0 {
		return nil
	}
	_, ipnet, err := net.ParseCIDR(podList.Items[0].Status.PodIP + "/16")

	if err != nil {
		return nil
	}
	d.cpClusterCIDR = ipnet
	return d.cpClusterCIDR
}
