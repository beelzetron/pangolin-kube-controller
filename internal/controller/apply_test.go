package controller

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"

	"pangolin-kube-controller/internal/config"
)

// fakeResourceClient implements the minimal ResourceClient used by deleteImmediate
type fakeResourceClient struct {
	deleted map[string]bool
	delErr  error
}

func (f *fakeResourceClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (f *fakeResourceClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if f.delErr != nil {
		return f.delErr
	}
	if f.deleted == nil {
		f.deleted = map[string]bool{}
	}
	f.deleted[name] = true
	return nil
}
func (f *fakeResourceClient) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return &unstructured.UnstructuredList{}, nil
}
func (f *fakeResourceClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func TestBuildDesiredSet(t *testing.T) {
	// adapt to expected type: map[string]json.RawMessage in production
	objs := make(map[string]json.RawMessage)
	objs["a"] = json.RawMessage("{}")
	objs["b"] = json.RawMessage("{}")
	desired := buildDesiredSet(objs)
	require.Len(t, desired, 2)
	require.Contains(t, desired, "a")
	require.Contains(t, desired, "b")
}

func TestStaleManagedName(t *testing.T) {
	cfg := &config.Config{
		ManagedAnnoKey:   "managed-by",
		ManagedAnnoValue: "pangolin",
	}
	c := &Controller{cfg: cfg}

	// case: present in desired -> not stale
	obj := unstructured.Unstructured{}
	obj.SetName("keep")
	obj.SetAnnotations(map[string]string{"managed-by": "pangolin"})
	desired := map[string]struct{}{"keep": {}}
	name, stale := c.staleManagedName(obj, desired)
	require.Equal(t, "keep", name)
	require.False(t, stale)

	// case: not in desired, annotation mismatch -> not stale
	obj2 := unstructured.Unstructured{}
	obj2.SetName("other")
	obj2.SetAnnotations(map[string]string{"managed-by": "someone-else"})
	name2, stale2 := c.staleManagedName(obj2, map[string]struct{}{})
	require.Equal(t, "other", name2)
	require.False(t, stale2)

	// case: not in desired, annotation matches -> stale
	obj3 := unstructured.Unstructured{}
	obj3.SetName("stale")
	obj3.SetAnnotations(map[string]string{"managed-by": "pangolin"})
	name3, stale3 := c.staleManagedName(obj3, map[string]struct{}{})
	require.Equal(t, "stale", name3)
	require.True(t, stale3)
}

func TestDeleteImmediate(t *testing.T) {
	ctx := context.Background()
	// success
	f := &fakeResourceClient{}
	err := deleteImmediate(ctx, f, "foo")
	require.NoError(t, err)
	require.True(t, f.deleted["foo"])

	// not found should be treated as success
	nf := &fakeResourceClient{delErr: kerrors.NewNotFound(schema.GroupResource{Group: "x", Resource: "y"}, "missing")}
	err = deleteImmediate(ctx, nf, "missing")
	require.NoError(t, err)

	// other error should be returned
	boom := &fakeResourceClient{delErr: errors.New("boom")}
	err = deleteImmediate(ctx, boom, "bad")
	require.Error(t, err)
}
