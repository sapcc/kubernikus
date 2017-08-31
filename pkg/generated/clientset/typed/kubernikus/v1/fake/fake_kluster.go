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

// FakeKlusters implements KlusterInterface
type FakeKlusters struct {
	Fake *FakeKubernikusV1
	ns   string
}

var klustersResource = schema.GroupVersionResource{Group: "kubernikus", Version: "v1", Resource: "klusters"}

var klustersKind = schema.GroupVersionKind{Group: "kubernikus", Version: "v1", Kind: "Kluster"}

// Get takes name of the kluster, and returns the corresponding kluster object, and an error if there is any.
func (c *FakeKlusters) Get(name string, options v1.GetOptions) (result *kubernikus_v1.Kluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(klustersResource, c.ns, name), &kubernikus_v1.Kluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.Kluster), err
}

// List takes label and field selectors, and returns the list of Klusters that match those selectors.
func (c *FakeKlusters) List(opts v1.ListOptions) (result *kubernikus_v1.KlusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(klustersResource, klustersKind, c.ns, opts), &kubernikus_v1.KlusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubernikus_v1.KlusterList{}
	for _, item := range obj.(*kubernikus_v1.KlusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested klusters.
func (c *FakeKlusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(klustersResource, c.ns, opts))

}

// Create takes the representation of a kluster and creates it.  Returns the server's representation of the kluster, and an error, if there is any.
func (c *FakeKlusters) Create(kluster *kubernikus_v1.Kluster) (result *kubernikus_v1.Kluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(klustersResource, c.ns, kluster), &kubernikus_v1.Kluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.Kluster), err
}

// Update takes the representation of a kluster and updates it. Returns the server's representation of the kluster, and an error, if there is any.
func (c *FakeKlusters) Update(kluster *kubernikus_v1.Kluster) (result *kubernikus_v1.Kluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(klustersResource, c.ns, kluster), &kubernikus_v1.Kluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.Kluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKlusters) UpdateStatus(kluster *kubernikus_v1.Kluster) (*kubernikus_v1.Kluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(klustersResource, "status", c.ns, kluster), &kubernikus_v1.Kluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.Kluster), err
}

// Delete takes name of the kluster and deletes it. Returns an error if one occurs.
func (c *FakeKlusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(klustersResource, c.ns, name), &kubernikus_v1.Kluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKlusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(klustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &kubernikus_v1.KlusterList{})
	return err
}

// Patch applies the patch and returns the patched kluster.
func (c *FakeKlusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubernikus_v1.Kluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(klustersResource, c.ns, name, data, subresources...), &kubernikus_v1.Kluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubernikus_v1.Kluster), err
}
