package apply

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"pangolin-kube-controller/internal/kube/resources"
)

func TestIngressRouteOpsApplyReadOnly(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		ManagedAnnoKey:    AnnoKey,
		ManagedAnnoValue:  AnnoVal,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          true,
	}

	u := map[string]interface{}{
		"metadata": map[string]interface{}{"name": TestRoute},
	}

	err := ops.Apply(context.Background(), TestRoute, u)
	require.NoError(t, err, "ReadOnly mode should return no error and not create resource")
	require.Equal(t, 0, fakeClient.createCount, "Create should not be called in ReadOnly mode")
}

func TestIngressRouteOpsApplyUpdatesExisting(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{
		existing: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":            TestRoute,
					"namespace":       TestNS,
					"resourceVersion": "1",
				},
			},
		},
	}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		ManagedAnnoKey:    AnnoKey,
		ManagedAnnoValue:  AnnoVal,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          false,
	}

	u := map[string]interface{}{
		"metadata": map[string]interface{}{"name": TestRoute},
	}

	err := ops.Apply(context.Background(), TestRoute, u)
	require.NoError(t, err, "Apply should succeed for existing resource")
	require.Equal(t, 1, fakeClient.patchCount, "Patch should be called once")
}

func TestIngressRouteOpsApplyGetError(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{getErr: errors.New(GetError)}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		ReadOnly:          false,
	}

	u := map[string]interface{}{}

	err := ops.Apply(context.Background(), TestRoute, u)
	require.Error(t, err, "Get error should be returned")
	require.Contains(t, err.Error(), GetError)
}

func TestIngressRouteOpsApplySingleReadOnly(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          true,
	}

	m := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "test-kind"},
	}

	err := ops.ApplySingle(context.Background(), m, "Middleware")
	require.NoError(t, err, "ReadOnly should return no error")
	require.Equal(t, 0, fakeClient.createCount, "Create should not be called")
}

func TestIngressRouteOpsApplySingleWithEmptyName(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          false,
	}

	m := map[string]interface{}{
		"metadata": map[string]interface{}{"name": ""},
	}

	err := ops.ApplySingle(context.Background(), m, "Middleware")
	require.NoError(t, err, "Empty name should return nil without calling client")
	require.Equal(t, 0, fakeClient.createCount, "Create should not be called for empty name")
}

func TestIngressRouteOpsApplySingleUpdatesExisting(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{
		existing: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":            TestMW,
					"namespace":       TestNS,
					"resourceVersion": "1",
				},
			},
		},
	}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          false,
	}

	m := map[string]interface{}{
		"metadata": map[string]interface{}{"name": TestMW},
	}

	err := ops.ApplySingle(context.Background(), m, "Middleware")
	require.NoError(t, err, "ApplySingle should succeed for existing resource")
	require.Equal(t, 1, fakeClient.patchCount, "Patch should be called")
}

func TestIngressRouteOpsApplySingleGetError(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeResourceClient{getErr: errors.New(GetError)}
	ops := &IngressRouteOps{
		ResIfc:            fakeClient,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		IngressClass:      TraefikIngressClass,
		ReadOnly:          false,
	}

	m := map[string]interface{}{
		"metadata": map[string]interface{}{"name": "test-mw"},
	}

	err := ops.ApplySingle(context.Background(), m, "Middleware")
	require.Error(t, err, "Get error should be returned")
	require.Contains(t, err.Error(), GetError)
}

type fakeResourceClient struct {
	resources.ResourceClient
	existing    *unstructured.Unstructured
	getErr      error
	createErr   error
	patchErr    error
	createCount int
	patchCount  int
}

func (c *fakeResourceClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*unstructured.Unstructured, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	if c.existing != nil {
		return c.existing, nil
	}
	return nil, k8serrors.NewNotFound(schema.GroupResource{Group: "traefik.io", Resource: "ingressroute"}, name)
}

func (c *fakeResourceClient) Create(_ context.Context, obj *unstructured.Unstructured, _ metav1.CreateOptions) (*unstructured.Unstructured, error) {
	c.createCount++
	if c.createErr != nil {
		return nil, c.createErr
	}
	return obj, nil
}

func (c *fakeResourceClient) Patch(_ context.Context, name string, _ types.PatchType, _ []byte, _ metav1.PatchOptions) (*unstructured.Unstructured, error) {
	c.patchCount++
	if c.patchErr != nil {
		return nil, c.patchErr
	}
	if c.existing != nil {
		return c.existing, nil
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{"metadata": map[string]interface{}{"name": name}}}, nil
}

func (*fakeResourceClient) Delete(_ context.Context, _ string, _ metav1.DeleteOptions) error {
	return nil
}

func (*fakeResourceClient) List(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, nil
}

func (*fakeResourceClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}
