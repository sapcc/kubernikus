package fake

import (
	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSAPCCloudProviderConfigs implements SAPCCloudProviderConfigInterface
type FakeSAPCCloudProviderConfigs struct {
	Fake *FakeKubernikusV1
	ns   string
}

var sapccloudproviderconfigsResource = schema.GroupVersionResource{Group: "kubernikus.sap.cc", Version: "v1", Resource: "sapccloudproviderconfigs"}

var sapccloudproviderconfigsKind = schema.GroupVersionKind{Group: "kubernikus.sap.cc", Version: "v1", Kind: "SAPCCloudProviderConfig"}

// Get takes name of the sAPCCloudProviderConfig, and returns the corresponding sAPCCloudProviderConfig object, and an error if there is any.
func (c *FakeSAPCCloudProviderConfigs) Get(name string, options v1.GetOptions) (result *kubernikus_v1.SAPCCloudProviderConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(sapccloudproviderconfigsResource, c.ns, name), &kubernikus_v1.SAPCCloudProviderConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.SAPCCloudProviderConfig), err
}

// List takes label and field selectors, and returns the list of SAPCCloudProviderConfigs that match those selectors.
func (c *FakeSAPCCloudProviderConfigs) List(opts v1.ListOptions) (result *kubernikus_v1.SAPCCloudProviderConfigList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(sapccloudproviderconfigsResource, sapccloudproviderconfigsKind, c.ns, opts), &kubernikus_v1.SAPCCloudProviderConfigList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubernikus_v1.SAPCCloudProviderConfigList{}
	for _, item := range obj.(*kubernikus_v1.SAPCCloudProviderConfigList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested sAPCCloudProviderConfigs.
func (c *FakeSAPCCloudProviderConfigs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(sapccloudproviderconfigsResource, c.ns, opts))

}

// Create takes the representation of a sAPCCloudProviderConfig and creates it.  Returns the server's representation of the sAPCCloudProviderConfig, and an error, if there is any.
func (c *FakeSAPCCloudProviderConfigs) Create(sAPCCloudProviderConfig *kubernikus_v1.SAPCCloudProviderConfig) (result *kubernikus_v1.SAPCCloudProviderConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(sapccloudproviderconfigsResource, c.ns, sAPCCloudProviderConfig), &kubernikus_v1.SAPCCloudProviderConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.SAPCCloudProviderConfig), err
}

// Update takes the representation of a sAPCCloudProviderConfig and updates it. Returns the server's representation of the sAPCCloudProviderConfig, and an error, if there is any.
func (c *FakeSAPCCloudProviderConfigs) Update(sAPCCloudProviderConfig *kubernikus_v1.SAPCCloudProviderConfig) (result *kubernikus_v1.SAPCCloudProviderConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(sapccloudproviderconfigsResource, c.ns, sAPCCloudProviderConfig), &kubernikus_v1.SAPCCloudProviderConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.SAPCCloudProviderConfig), err
}

// Delete takes name of the sAPCCloudProviderConfig and deletes it. Returns an error if one occurs.
func (c *FakeSAPCCloudProviderConfigs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(sapccloudproviderconfigsResource, c.ns, name), &kubernikus_v1.SAPCCloudProviderConfig{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSAPCCloudProviderConfigs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(sapccloudproviderconfigsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &kubernikus_v1.SAPCCloudProviderConfigList{})
	return err
}

// Patch applies the patch and returns the patched sAPCCloudProviderConfig.
func (c *FakeSAPCCloudProviderConfigs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubernikus_v1.SAPCCloudProviderConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(sapccloudproviderconfigsResource, c.ns, name, data, subresources...), &kubernikus_v1.SAPCCloudProviderConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.SAPCCloudProviderConfig), err
}
