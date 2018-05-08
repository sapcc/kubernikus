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

// FakeExternalNodes implements ExternalNodeInterface
type FakeExternalNodes struct {
	Fake *FakeKubernikusV1
	ns   string
}

var externalnodesResource = schema.GroupVersionResource{Group: "kubernikus.sap.cc", Version: "v1", Resource: "externalnodes"}

var externalnodesKind = schema.GroupVersionKind{Group: "kubernikus.sap.cc", Version: "v1", Kind: "ExternalNode"}

// Get takes name of the externalNode, and returns the corresponding externalNode object, and an error if there is any.
func (c *FakeExternalNodes) Get(name string, options v1.GetOptions) (result *kubernikus_v1.ExternalNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(externalnodesResource, c.ns, name), &kubernikus_v1.ExternalNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.ExternalNode), err
}

// List takes label and field selectors, and returns the list of ExternalNodes that match those selectors.
func (c *FakeExternalNodes) List(opts v1.ListOptions) (result *kubernikus_v1.ExternalNodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(externalnodesResource, externalnodesKind, c.ns, opts), &kubernikus_v1.ExternalNodeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubernikus_v1.ExternalNodeList{}
	for _, item := range obj.(*kubernikus_v1.ExternalNodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested externalNodes.
func (c *FakeExternalNodes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(externalnodesResource, c.ns, opts))

}

// Create takes the representation of a externalNode and creates it.  Returns the server's representation of the externalNode, and an error, if there is any.
func (c *FakeExternalNodes) Create(externalNode *kubernikus_v1.ExternalNode) (result *kubernikus_v1.ExternalNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(externalnodesResource, c.ns, externalNode), &kubernikus_v1.ExternalNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.ExternalNode), err
}

// Update takes the representation of a externalNode and updates it. Returns the server's representation of the externalNode, and an error, if there is any.
func (c *FakeExternalNodes) Update(externalNode *kubernikus_v1.ExternalNode) (result *kubernikus_v1.ExternalNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(externalnodesResource, c.ns, externalNode), &kubernikus_v1.ExternalNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.ExternalNode), err
}

// Delete takes name of the externalNode and deletes it. Returns an error if one occurs.
func (c *FakeExternalNodes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(externalnodesResource, c.ns, name), &kubernikus_v1.ExternalNode{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeExternalNodes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(externalnodesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &kubernikus_v1.ExternalNodeList{})
	return err
}

// Patch applies the patch and returns the patched externalNode.
func (c *FakeExternalNodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubernikus_v1.ExternalNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(externalnodesResource, c.ns, name, data, subresources...), &kubernikus_v1.ExternalNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.ExternalNode), err
}
