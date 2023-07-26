package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/validate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

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

	if params.Body.Spec.Version != "" {
		selectedVersion, found := d.Images.Versions[params.Body.Spec.Version]
		if !found || !selectedVersion.Supported {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 400, "Specified cluster version %s not supported", params.Body.Spec.Version)
		}
	}

	spec.Name = name
	for i, pool := range spec.NodePools {
		// Set default image
		if pool.Image == "" {
			spec.NodePools[i].Image = DEFAULT_IMAGE
		}

		allowReboot := true
		allowReplace := true
		if pool.Config == nil {
			spec.NodePools[i].Config = &models.NodePoolConfig{
				AllowReboot:  &allowReboot,
				AllowReplace: &allowReplace,
			}
		}

		if spec.NodePools[i].Config.AllowReboot == nil {
			spec.NodePools[i].Config.AllowReboot = &allowReboot
		}

		if spec.NodePools[i].Config.AllowReplace == nil {
			spec.NodePools[i].Config.AllowReplace = &allowReplace
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

	if kluster.ClusterCIDR() == "" && !kluster.Spec.NoCloud {
		return NewErrorResponse(&operations.CreateClusterDefault{}, 400, "Specifying an empty ClusterCIDR is only allowed with noCloud: true")
	}

	//Ensure that the service CIDR range does not overlap with any control plane CIDR
	//Otherwise the wormhole server will prevent the kluster apiserver from functioning properly
	if overlap, err := d.overlapWithControlPlane(kluster.Spec.ServiceCIDR); overlap {
		return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Service CIDR %s not allowed: %s", kluster.Spec.ServiceCIDR, err)
	}

	if kluster.ClusterCIDR() != "" {
		//Ensure that the cluster CIDR range does not overlap with any control plane CIDR
		//Otherwise the wormhole server will prevent the kluster apiserver from functioning properly
		if overlap, err := d.overlapWithControlPlane(*kluster.Spec.ClusterCIDR); overlap {
			return NewErrorResponse(&operations.CreateClusterDefault{}, 409, "Cluster CIDR %s not allowed: %s", kluster.ClusterCIDR(), err)
		}

		//Ensure that the clust CIDR range does not overlap with other clusters in the same project
		if !kluster.Spec.NoCloud {
			if overlap, err := d.overlapWithSiblingCluster(kluster.ClusterCIDR(), kluster.Spec.Openstack.RouterID, principal); overlap || err != nil {
				if overlap {
					return NewErrorResponse(&operations.CreateClusterDefault{}, 409, err.Error())
				}
				return NewErrorResponse(&operations.CreateClusterDefault{}, 500, err.Error())
			}
		}
	}

	kluster.ObjectMeta = metav1.ObjectMeta{
		Name: qualifiedName(name, principal.Account),
		Labels: map[string]string{
			"account":                             principal.Account,
			"kubernikus.cloud.sap/seed-reconcile": "true",
		},
		Annotations: map[string]string{"creator": fmt.Sprintf("%s/%s", principal.Name, principal.Domain)},
	}

	k8sutil.EnsureNamespace(d.Kubernetes, d.Namespace)
	kluster, err = d.Kubernikus.KubernikusV1().Klusters(d.Namespace).Create(context.TODO(), kluster, metav1.CreateOptions{})
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

	//Wait for a second so that the newly created cluster shows up in the cache
	//This is a hack so that a subsequent GET /api/v1/cluster/:name will not return 404
	wait.Poll(50*time.Millisecond, 2*time.Second, func() (bool, error) { //nolint:staticcheck
		if _, err := d.Klusters.Klusters(d.Namespace).Get(kluster.Name); err != nil {
			return false, nil
		}
		return true, nil
	})

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

func (d *createCluster) overlapWithSiblingCluster(cidr string, routerID string, principal *models.Principal) (bool, error) {
	listOpts := metav1.ListOptions{LabelSelector: accountSelector(principal).String()}
	klusterList, err := d.Kubernikus.KubernikusV1().Klusters(d.Namespace).List(context.TODO(), listOpts)
	if err != nil {
		return false, err
	}
	for _, other := range klusterList.Items {
		if other.ClusterCIDR() == "" {
			continue
		}
		if routerID == "" || routerID == other.Spec.Openstack.RouterID {
			_, ourCIDR, err := net.ParseCIDR(cidr)
			if err != nil {
				return false, err
			}
			_, otherCIDR, err := net.ParseCIDR(*other.Spec.ClusterCIDR)
			if err != nil {
				return false, err
			}
			if ip.CIDROverlap(ourCIDR, otherCIDR) {
				return true, fmt.Errorf("Cluster CIDR %s overlaps with cluster CIDR %s from cluster '%s'. Specify a different CIDR Range or use a dedicated router for this cluster", cidr, other.ClusterCIDR(), other.Spec.Name)
			}
		}
	}
	return false, nil
}

// approximate the control plane service CIDR by getting one service IP and assuming a /17 prefix
func (d *createCluster) controlPlaneServiceCIDR() *net.IPNet {
	if d.cpServiceCIDR != nil {
		return d.cpServiceCIDR
	}
	svc, err := d.Kubernetes.CoreV1().Services("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
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

// we infer the clusterCIDR by taking a Pod IP and assuming /16
func (d *createCluster) controlPlaneClusterCIDR() *net.IPNet {
	if d.cpClusterCIDR != nil {
		return d.cpClusterCIDR
	}
	podList, err := d.Kubernetes.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})

	if err != nil || len(podList.Items) == 0 {
		return nil
	}
	//find pod which is not in hostNetwork mode
	idx := 0
	for idx = range podList.Items {
		if !podList.Items[idx].Spec.HostNetwork {
			break
		}
	}
	_, ipnet, err := net.ParseCIDR(podList.Items[idx].Status.PodIP + "/16")

	if err != nil {
		return nil
	}
	d.cpClusterCIDR = ipnet
	return d.cpClusterCIDR
}
