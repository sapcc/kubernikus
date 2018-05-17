package v1

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	scheme "github.com/sapcc/kubernikus/pkg/generated/clientset/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SAPCCloudProviderConfigsGetter has a method to return a SAPCCloudProviderConfigInterface.
// A group's client should implement this interface.
type SAPCCloudProviderConfigsGetter interface {
	SAPCCloudProviderConfigs(namespace string) SAPCCloudProviderConfigInterface
}

// SAPCCloudProviderConfigInterface has methods to work with SAPCCloudProviderConfig resources.
type SAPCCloudProviderConfigInterface interface {
	Create(*v1.SAPCCloudProviderConfig) (*v1.SAPCCloudProviderConfig, error)
	Update(*v1.SAPCCloudProviderConfig) (*v1.SAPCCloudProviderConfig, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.SAPCCloudProviderConfig, error)
	List(opts meta_v1.ListOptions) (*v1.SAPCCloudProviderConfigList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SAPCCloudProviderConfig, err error)
	SAPCCloudProviderConfigExpansion
}

// sAPCCloudProviderConfigs implements SAPCCloudProviderConfigInterface
type sAPCCloudProviderConfigs struct {
	client rest.Interface
	ns     string
}

// newSAPCCloudProviderConfigs returns a SAPCCloudProviderConfigs
func newSAPCCloudProviderConfigs(c *KubernikusV1Client, namespace string) *sAPCCloudProviderConfigs {
	return &sAPCCloudProviderConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the sAPCCloudProviderConfig, and returns the corresponding sAPCCloudProviderConfig object, and an error if there is any.
func (c *sAPCCloudProviderConfigs) Get(name string, options meta_v1.GetOptions) (result *v1.SAPCCloudProviderConfig, err error) {
	result = &v1.SAPCCloudProviderConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SAPCCloudProviderConfigs that match those selectors.
func (c *sAPCCloudProviderConfigs) List(opts meta_v1.ListOptions) (result *v1.SAPCCloudProviderConfigList, err error) {
	result = &v1.SAPCCloudProviderConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested sAPCCloudProviderConfigs.
func (c *sAPCCloudProviderConfigs) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a sAPCCloudProviderConfig and creates it.  Returns the server's representation of the sAPCCloudProviderConfig, and an error, if there is any.
func (c *sAPCCloudProviderConfigs) Create(sAPCCloudProviderConfig *v1.SAPCCloudProviderConfig) (result *v1.SAPCCloudProviderConfig, err error) {
	result = &v1.SAPCCloudProviderConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		Body(sAPCCloudProviderConfig).
		Do().
		Into(result)
	return
}

// Update takes the representation of a sAPCCloudProviderConfig and updates it. Returns the server's representation of the sAPCCloudProviderConfig, and an error, if there is any.
func (c *sAPCCloudProviderConfigs) Update(sAPCCloudProviderConfig *v1.SAPCCloudProviderConfig) (result *v1.SAPCCloudProviderConfig, err error) {
	result = &v1.SAPCCloudProviderConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		Name(sAPCCloudProviderConfig.Name).
		Body(sAPCCloudProviderConfig).
		Do().
		Into(result)
	return
}

// Delete takes name of the sAPCCloudProviderConfig and deletes it. Returns an error if one occurs.
func (c *sAPCCloudProviderConfigs) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sAPCCloudProviderConfigs) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched sAPCCloudProviderConfig.
func (c *sAPCCloudProviderConfigs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SAPCCloudProviderConfig, err error) {
	result = &v1.SAPCCloudProviderConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("sapccloudproviderconfigs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
