package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekube "k8s.io/client-go/kubernetes/fake"
)

func TestEndpointSliceOpsApplyReadOnly(t *testing.T) {
	t.Parallel()

	cli := fakekube.NewClientset()
	ops := &EndpointSliceOps{
		Kube:              cli,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedLabelKeyFull,
		ManagedLabelValue: ManagedLabelValueController,
		ReadOnly:          true,
	}

	es := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: TestESName},
		Endpoints:   []discoveryv1.Endpoint{},
		Ports:       []discoveryv1.EndpointPort{},
		AddressType: discoveryv1.AddressTypeIPv4,
	}

	err := ops.Apply(context.Background(), es)
	require.NoError(t, err, "ReadOnly mode should return no error")
}

func TestEndpointSliceOpsApplyCreate(t *testing.T) {
	t.Parallel()

	cli := fakekube.NewClientset()
	ops := &EndpointSliceOps{
		Kube:                      cli,
		Namespace:                 TestNS,
		ManagedLabelKey:           ManagedLabelKeyFull,
		ManagedLabelValue:         ManagedLabelValueController,
		ManagedAnnoKey:            ManagedAnnoKeyPangolin,
		ManagedAnnoValue:          ManagedLabelValueController,
		TraefikInstanceLabelKey:   InstanceLabelKeyFull,
		TraefikInstanceLabelValue: InstanceLabelValueMyInstance,
		ReadOnly:                  false,
	}

	es := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: NewES},
		Endpoints:   []discoveryv1.Endpoint{},
		Ports:       []discoveryv1.EndpointPort{},
		AddressType: discoveryv1.AddressTypeIPv4,
	}

	err := ops.Apply(context.Background(), es)
	require.NoError(t, err, "Create should succeed")

	created, err := cli.DiscoveryV1().EndpointSlices(TestNS).Get(context.Background(), NewES, metav1.GetOptions{})
	require.NoError(t, err, "Created EndpointSlice should be retrievable")
	require.Equal(t, ManagedLabelValueController, created.Labels[ManagedLabelKeyFull])
	require.Equal(t, InstanceLabelValueMyInstance, created.Labels[InstanceLabelKeyFull])
	require.Equal(t, ManagedLabelValueController, created.Annotations[ManagedAnnoKeyPangolin])
}

func TestEndpointSliceOpsApplyUpdate(t *testing.T) {
	t.Parallel()

	existing := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ExistingES,
			Namespace: TestNS,
			Labels:    map[string]string{},
		},
		Endpoints:   []discoveryv1.Endpoint{},
		Ports:       []discoveryv1.EndpointPort{},
		AddressType: discoveryv1.AddressTypeIPv4,
	}
	cli := fakekube.NewClientset(existing)
	ops := &EndpointSliceOps{
		Kube:              cli,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedLabelKeyFull,
		ManagedLabelValue: ManagedLabelValueController,
		ReadOnly:          false,
	}

	newEs := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: ExistingES},
		Endpoints:   []discoveryv1.Endpoint{{}},
		Ports:       []discoveryv1.EndpointPort{{Port: int32Ptr(8080)}},
		AddressType: discoveryv1.AddressTypeIPv4,
	}

	err := ops.Apply(context.Background(), newEs)
	require.NoError(t, err, "Update should succeed")

	updated, err := cli.DiscoveryV1().EndpointSlices(TestNS).Get(context.Background(), ExistingES, metav1.GetOptions{})
	require.NoError(t, err, "Updated EndpointSlice should be retrievable")
	require.Equal(t, 1, len(updated.Endpoints), "Endpoints should be updated")
	require.Equal(t, 1, len(updated.Ports), "Ports should be updated")
	require.Equal(t, ManagedLabelValueController, updated.Labels[ManagedLabelKeyFull], "Managed label should be set")
}

func TestEndpointSliceOpsApplyUpdateRemovesPreviouslyManagedKeys(t *testing.T) {
	t.Parallel()

	const keepExternalKey = "keep-external"

	existing := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ExistingES,
			Namespace: TestNS,
			Labels: map[string]string{
				keepExternalKey: "1",
				"to-remove":     "x",
			},
			Annotations: map[string]string{
				managedLabelKeysAnnotation: "[\"to-remove\"]",
				managedAnnoKeysAnnotation:  "[\"anno-remove\"]",
				"anno-remove":              "y",
			},
		},
		Endpoints:   []discoveryv1.Endpoint{},
		Ports:       []discoveryv1.EndpointPort{},
		AddressType: discoveryv1.AddressTypeIPv4,
	}

	cli := fakekube.NewClientset(existing)
	ops := &EndpointSliceOps{
		Kube:              cli,
		Namespace:         TestNS,
		ManagedLabelKey:   ManagedLabelKeyFull,
		ManagedLabelValue: ManagedLabelValueController,
		ManagedAnnoKey:    ManagedAnnoKeyPangolin,
		ManagedAnnoValue:  ManagedLabelValueController,
		ReadOnly:          false,
	}

	newEs := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: ExistingES,
			Labels: map[string]string{
				keepExternalKey: "1",
				// intentionally omit "to-remove"
			},
			Annotations: map[string]string{
				// intentionally omit "anno-remove"
			},
		},
		Endpoints:   []discoveryv1.Endpoint{{}},
		Ports:       []discoveryv1.EndpointPort{{Port: int32Ptr(8080)}},
		AddressType: discoveryv1.AddressTypeIPv4,
	}

	err := ops.Apply(context.Background(), newEs)
	require.NoError(t, err, "Update should succeed")

	updated, err := cli.DiscoveryV1().EndpointSlices(TestNS).Get(context.Background(), ExistingES, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, updated.Labels, "to-remove")
	require.Equal(t, "1", updated.Labels[keepExternalKey], "external label should remain")
	require.NotContains(t, updated.Annotations, "anno-remove")
	require.Equal(t, ManagedLabelValueController, updated.Annotations[ManagedAnnoKeyPangolin], "managed annotation should be present")
}

func TestEnsureManagedMeta(t *testing.T) {
	t.Parallel()

	ops := &EndpointSliceOps{
		ManagedLabelKey:   ManagedBy,
		ManagedLabelValue: Controller,
		ManagedAnnoKey:    AnnoKey,
		ManagedAnnoValue:  AnnoVal,
	}

	es := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{},
	}

	ops.ensureManagedMeta(es)

	require.Equal(t, Controller, es.Labels[ManagedBy])
	require.Equal(t, AnnoVal, es.Annotations[AnnoKey])
}

func TestEnsureManagedMetaWithInstanceLabel(t *testing.T) {
	t.Parallel()

	ops := &EndpointSliceOps{
		ManagedLabelKey:           ManagedBy,
		ManagedLabelValue:         Controller,
		ManagedAnnoKey:            AnnoKey,
		ManagedAnnoValue:          AnnoVal,
		TraefikInstanceLabelKey:   InstanceKey,
		TraefikInstanceLabelValue: InstanceLabelValueMyInstance,
	}

	es := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{},
	}

	ops.ensureManagedMeta(es)

	require.Equal(t, Controller, es.Labels[ManagedBy])
	require.Equal(t, InstanceLabelValueMyInstance, es.Labels[InstanceKey])
	require.Equal(t, AnnoVal, es.Annotations[AnnoKey])
}

func TestEnsureManagedMetaNilLabels(t *testing.T) {
	t.Parallel()

	ops := &EndpointSliceOps{
		ManagedLabelKey:           ManagedBy,
		ManagedLabelValue:         Controller,
		TraefikInstanceLabelKey:   InstanceKey,
		TraefikInstanceLabelValue: InstanceLabelValueMyInstance,
	}

	es := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Labels: nil, Annotations: nil},
	}

	ops.ensureManagedMeta(es)

	require.NotNil(t, es.Labels)
	require.Equal(t, Controller, es.Labels[ManagedBy])
	require.Equal(t, InstanceLabelValueMyInstance, es.Labels[InstanceKey])
	require.NotNil(t, es.Annotations)
}

func int32Ptr(v int32) *int32 { return &v }
