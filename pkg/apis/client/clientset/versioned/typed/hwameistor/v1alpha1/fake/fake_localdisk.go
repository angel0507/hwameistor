// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeLocalDisks implements LocalDiskInterface
type FakeLocalDisks struct {
	Fake *FakeHwameistorV1alpha1
}

var localdisksResource = schema.GroupVersionResource{Group: "hwameistor.io", Version: "v1alpha1", Resource: "localdisks"}

var localdisksKind = schema.GroupVersionKind{Group: "hwameistor.io", Version: "v1alpha1", Kind: "LocalDisk"}

// Get takes name of the localDisk, and returns the corresponding localDisk object, and an error if there is any.
func (c *FakeLocalDisks) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.LocalDisk, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(localdisksResource, name), &v1alpha1.LocalDisk{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LocalDisk), err
}

// List takes label and field selectors, and returns the list of LocalDisks that match those selectors.
func (c *FakeLocalDisks) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.LocalDiskList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(localdisksResource, localdisksKind, opts), &v1alpha1.LocalDiskList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.LocalDiskList{ListMeta: obj.(*v1alpha1.LocalDiskList).ListMeta}
	for _, item := range obj.(*v1alpha1.LocalDiskList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested localDisks.
func (c *FakeLocalDisks) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(localdisksResource, opts))
}

// Create takes the representation of a localDisk and creates it.  Returns the server's representation of the localDisk, and an error, if there is any.
func (c *FakeLocalDisks) Create(ctx context.Context, localDisk *v1alpha1.LocalDisk, opts v1.CreateOptions) (result *v1alpha1.LocalDisk, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(localdisksResource, localDisk), &v1alpha1.LocalDisk{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LocalDisk), err
}

// Update takes the representation of a localDisk and updates it. Returns the server's representation of the localDisk, and an error, if there is any.
func (c *FakeLocalDisks) Update(ctx context.Context, localDisk *v1alpha1.LocalDisk, opts v1.UpdateOptions) (result *v1alpha1.LocalDisk, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(localdisksResource, localDisk), &v1alpha1.LocalDisk{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LocalDisk), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeLocalDisks) UpdateStatus(ctx context.Context, localDisk *v1alpha1.LocalDisk, opts v1.UpdateOptions) (*v1alpha1.LocalDisk, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(localdisksResource, "status", localDisk), &v1alpha1.LocalDisk{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LocalDisk), err
}

// Delete takes name of the localDisk and deletes it. Returns an error if one occurs.
func (c *FakeLocalDisks) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(localdisksResource, name), &v1alpha1.LocalDisk{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeLocalDisks) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(localdisksResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.LocalDiskList{})
	return err
}

// Patch applies the patch and returns the patched localDisk.
func (c *FakeLocalDisks) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.LocalDisk, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(localdisksResource, name, pt, data, subresources...), &v1alpha1.LocalDisk{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LocalDisk), err
}
