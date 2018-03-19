package v1

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	scheme "github.com/sapcc/kubernikus/pkg/generated/clientset/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ExternalNodesGetter has a method to return a ExternalNodeInterface.
// A group's client should implement this interface.
type ExternalNodesGetter interface {
	ExternalNodes(namespace string) ExternalNodeInterface
}

// ExternalNodeInterface has methods to work with ExternalNode resources.
type ExternalNodeInterface interface {
	Create(*v1.ExternalNode) (*v1.ExternalNode, error)
	Update(*v1.ExternalNode) (*v1.ExternalNode, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.ExternalNode, error)
	List(opts meta_v1.ListOptions) (*v1.ExternalNodeList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ExternalNode, err error)
	ExternalNodeExpansion
}

// externalNodes implements ExternalNodeInterface
type externalNodes struct {
	client rest.Interface
	ns     string
}

// newExternalNodes returns a ExternalNodes
func newExternalNodes(c *KubernikusV1Client, namespace string) *externalNodes {
	return &externalNodes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the externalNode, and returns the corresponding externalNode object, and an error if there is any.
func (c *externalNodes) Get(name string, options meta_v1.GetOptions) (result *v1.ExternalNode, err error) {
	result = &v1.ExternalNode{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("externalnodes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ExternalNodes that match those selectors.
func (c *externalNodes) List(opts meta_v1.ListOptions) (result *v1.ExternalNodeList, err error) {
	result = &v1.ExternalNodeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("externalnodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested externalNodes.
func (c *externalNodes) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("externalnodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a externalNode and creates it.  Returns the server's representation of the externalNode, and an error, if there is any.
func (c *externalNodes) Create(externalNode *v1.ExternalNode) (result *v1.ExternalNode, err error) {
	result = &v1.ExternalNode{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("externalnodes").
		Body(externalNode).
		Do().
		Into(result)
	return
}

// Update takes the representation of a externalNode and updates it. Returns the server's representation of the externalNode, and an error, if there is any.
func (c *externalNodes) Update(externalNode *v1.ExternalNode) (result *v1.ExternalNode, err error) {
	result = &v1.ExternalNode{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("externalnodes").
		Name(externalNode.Name).
		Body(externalNode).
		Do().
		Into(result)
	return
}

// Delete takes name of the externalNode and deletes it. Returns an error if one occurs.
func (c *externalNodes) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("externalnodes").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *externalNodes) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("externalnodes").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched externalNode.
func (c *externalNodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ExternalNode, err error) {
	result = &v1.ExternalNode{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("externalnodes").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
