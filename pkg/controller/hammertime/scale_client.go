package hammertime

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

type DeploymentScaleClient interface {
	GetScale(deploymentName string) (scale int, err error)
	UpdateScale(deploymentName string, scale int) (err error)
}

type deploymentScaleClient struct {
	client rest.Interface
	ns     string
	appsv1 bool
}

func NewScaleClient(client kubernetes.Interface, namespace string) DeploymentScaleClient {
	if v, err := client.Discovery().ServerVersion(); err == nil {
		if parsedVersion, err := version.ParseGeneric(v.GitVersion); err == nil {
			if parsedVersion.Minor() > 15 {
				return &deploymentScaleClient{client.AppsV1().RESTClient(), namespace, true}
			}
		}
	}
	return &deploymentScaleClient{client.ExtensionsV1beta1().RESTClient(), namespace, false}
}

func (c *deploymentScaleClient) GetScale(deploymentName string) (scale int, err error) {

	result := c.client.Get().
		Namespace(c.ns).
		Resource("deployments").
		Name(deploymentName).
		SubResource("scale").
		Do()

	if c.appsv1 {
		r := &autoscalingv1.Scale{}
		return int(r.Spec.Replicas), result.Into(r)
	}
	r := &extensionsv1beta1.Scale{}
	return int(r.Spec.Replicas), result.Into(r)
}

func (c *deploymentScaleClient) UpdateScale(deploymentName string, replicas int) (err error) {
	var scale runtime.Object
	if c.appsv1 {
		scale = &autoscalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: c.ns}, Spec: autoscalingv1.ScaleSpec{Replicas: int32(replicas)}}
	} else {
		scale = &extensionsv1beta1.Scale{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: c.ns}, Spec: extensionsv1beta1.ScaleSpec{Replicas: int32(replicas)}}
	}
	return c.client.Put().
		Namespace(c.ns).
		Resource("deployments").
		Name(deploymentName).
		SubResource("scale").
		Body(scale).
		Do().Error()
}
