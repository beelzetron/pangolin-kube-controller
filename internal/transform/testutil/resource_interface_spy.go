package testutil

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// ResourceInterfaceSpy is a small test mock implementing a subset of
// dynamic.ResourceInterface. Some method parameters (context, name, opts)
// exist only to match the external interface and are intentionally unused.
type ResourceInterfaceSpy struct {
	LastCreateOptions metav1.CreateOptions
	LastPatchOptions  metav1.PatchOptions
}

func (r *ResourceInterfaceSpy) Create(_ context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions) (*unstructured.Unstructured, error) {
	r.LastCreateOptions = options
	return obj, nil
}

func (*ResourceInterfaceSpy) Update(context.Context, *unstructured.Unstructured, metav1.UpdateOptions, ...string) (*unstructured.Unstructured, error) {
	panic("unexpected call to Update")
}

func (*ResourceInterfaceSpy) UpdateStatus(context.Context, *unstructured.Unstructured, metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	panic("unexpected call to UpdateStatus")
}

func (*ResourceInterfaceSpy) Delete(context.Context, string, metav1.DeleteOptions) error {
	panic("unexpected call to Delete")
}

func (*ResourceInterfaceSpy) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	panic("unexpected call to DeleteCollection")
}

func (*ResourceInterfaceSpy) Get(context.Context, string, metav1.GetOptions) (*unstructured.Unstructured, error) {
	panic("unexpected call to Get")
}

func (*ResourceInterfaceSpy) List(context.Context, metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	panic("unexpected call to List")
}

func (*ResourceInterfaceSpy) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	panic("unexpected call to Watch")
}

func (r *ResourceInterfaceSpy) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, options metav1.PatchOptions) (*unstructured.Unstructured, error) {
	r.LastPatchOptions = options
	return &unstructured.Unstructured{}, nil
}

func (*ResourceInterfaceSpy) Apply(context.Context, string, *unstructured.Unstructured, metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	panic("unexpected call to Apply")
}

func (*ResourceInterfaceSpy) ApplyStatus(context.Context, string, *unstructured.Unstructured, metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	panic("unexpected call to ApplyStatus")
}
