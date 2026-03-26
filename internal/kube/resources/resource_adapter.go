package resources

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// ResourceClient is a small, project-local subset of dynamic.ResourceInterface
// used by the controller. It intentionally omits variadic subresources to keep
// signatures simpler for internal implementations and tests.
type ResourceClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error)
	Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type resourceAdapter struct {
	ri dynamic.ResourceInterface
}

// AdaptResource wraps a dynamic.ResourceInterface to expose only the subset of
// methods used by the controller, simplifying fakes and tests.
func AdaptResource(ri dynamic.ResourceInterface) ResourceClient { return resourceAdapter{ri: ri} }

func (a resourceAdapter) Get(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return a.ri.Get(ctx, name, opts)
}
func (a resourceAdapter) Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	return a.ri.Create(ctx, obj, opts)
}
func (a resourceAdapter) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error) {
	return a.ri.Patch(ctx, name, pt, data, opts)
}
func (a resourceAdapter) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return a.ri.Delete(ctx, name, opts)
}
func (a resourceAdapter) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return a.ri.List(ctx, opts)
}
func (a resourceAdapter) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return a.ri.Watch(ctx, opts)
}
