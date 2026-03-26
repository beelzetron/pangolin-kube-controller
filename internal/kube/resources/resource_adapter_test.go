package resources

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type fakeDynamicResource struct {
	getErr              error
	createErr           error
	updateErr           error
	updateStatusErr     error
	patchErr            error
	deleteErr           error
	deleteCollectionErr error
	listErr             error
	watchErr            error
	applyErr            error
	applyStatusErr      error
}

func (f *fakeDynamicResource) Get(_ context.Context, _ string, _ metav1.GetOptions, _ ...string) (*unstructured.Unstructured, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Create(_ context.Context, _ *unstructured.Unstructured, _ metav1.CreateOptions, _ ...string) (*unstructured.Unstructured, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (*unstructured.Unstructured, error) {
	if f.patchErr != nil {
		return nil, f.patchErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Update(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions, _ ...string) (*unstructured.Unstructured, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) UpdateStatus(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	if f.updateStatusErr != nil {
		return nil, f.updateStatusErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Delete(_ context.Context, _ string, _ metav1.DeleteOptions, _ ...string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

func (f *fakeDynamicResource) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	if f.deleteCollectionErr != nil {
		return f.deleteCollectionErr
	}
	return nil
}

func (f *fakeDynamicResource) List(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	if f.watchErr != nil {
		return nil, f.watchErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) Apply(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions, _ ...string) (*unstructured.Unstructured, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return nil, nil
}

func (f *fakeDynamicResource) ApplyStatus(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	if f.applyStatusErr != nil {
		return nil, f.applyStatusErr
	}
	return nil, nil
}

var _ dynamic.ResourceInterface = (*fakeDynamicResource)(nil)

func TestAdaptResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const testName = "test-name"

	t.Run("Get error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("get error")
		ri := &fakeDynamicResource{getErr: wantErr}
		client := AdaptResource(ri)
		_, got := client.Get(ctx, testName, metav1.GetOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("Get() error = %v, want %v", got, wantErr)
		}
	})

	t.Run("Create error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("create error")
		ri := &fakeDynamicResource{createErr: wantErr}
		client := AdaptResource(ri)
		_, got := client.Create(ctx, &unstructured.Unstructured{}, metav1.CreateOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("Create() error = %v, want %v", got, wantErr)
		}
	})

	t.Run("Patch error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("patch error")
		ri := &fakeDynamicResource{patchErr: wantErr}
		client := AdaptResource(ri)
		_, got := client.Patch(ctx, testName, types.JSONPatchType, []byte("{}"), metav1.PatchOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("Patch() error = %v, want %v", got, wantErr)
		}
	})

	t.Run("Delete error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("delete error")
		ri := &fakeDynamicResource{deleteErr: wantErr}
		client := AdaptResource(ri)
		got := client.Delete(ctx, testName, metav1.DeleteOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("Delete() error = %v, want %v", got, wantErr)
		}
	})

	t.Run("List error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("list error")
		ri := &fakeDynamicResource{listErr: wantErr}
		client := AdaptResource(ri)
		_, got := client.List(ctx, metav1.ListOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("List() error = %v, want %v", got, wantErr)
		}
	})

	t.Run("Watch error propagation", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("watch error")
		ri := &fakeDynamicResource{watchErr: wantErr}
		client := AdaptResource(ri)
		_, got := client.Watch(ctx, metav1.ListOptions{})
		if !errors.Is(got, wantErr) {
			t.Errorf("Watch() error = %v, want %v", got, wantErr)
		}
	})
}

func TestResourceClientInterface(t *testing.T) {
	t.Parallel()
	var _ ResourceClient = (*resourceAdapter)(nil)
}
