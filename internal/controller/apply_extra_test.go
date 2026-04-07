package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	fakekube "k8s.io/client-go/kubernetes/fake"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
	traefikconfig "pangolin-kube-controller/internal/transform/config"
	"pangolin-kube-controller/internal/transform/testutil"
)

// testCtxBg returns a plain background context for use in tests.
func testCtxBg() context.Context { return context.Background() }

// testCfgWithLabels returns a minimal Config with valid managed label fields so
// that label selectors built during GC are non-empty and valid.
func testCfgWithLabels() *config.Config {
	return &config.Config{
		ManagedLabelKey:   "app.kubernetes.io/managed-by",
		ManagedLabelValue: "pangolin",
	}
}

// TestGVRForKind verifies that gvrForKind returns the expected GVR for all
// known Traefik CRD kinds, and returns an empty GVR for unknown ones.
func TestGVRForKind(t *testing.T) {
	t.Parallel()

	c := NewController(testCfgWithLabels(), nil, nil, nil)

	tests := []struct {
		kind     string
		resource string
	}{
		{"Middleware", "middlewares"},
		{"IngressRoute", "ingressroutes"},
		{"TraefikService", "traefikservices"},
		{"ServersTransport", "serverstransports"},
		{"IngressRouteTCP", "ingressroutetcps"},
		{"IngressRouteUDP", "ingressrouteudps"},
		{"ServersTransportTCP", "serverstransporttcps"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.kind, func(t *testing.T) {
			t.Parallel()
			gvr := c.gvrForKind(tt.kind)
			require.Equal(t, traefikconfig.Group, gvr.Group)
			require.Equal(t, traefikconfig.Version, gvr.Version)
			require.Equal(t, tt.resource, gvr.Resource)
		})
	}
}

func TestGVRForKindUnknown(t *testing.T) {
	t.Parallel()

	c := NewController(testCfgWithLabels(), nil, nil, nil)
	gvr := c.gvrForKind("UnknownKind")
	require.Empty(t, gvr.Group)
	require.Empty(t, gvr.Version)
	require.Empty(t, gvr.Resource)
}

// TestApplyConfigForTestNilConfig exercises ApplyConfigForTest with a nil
// config (should return nil immediately).
func TestApplyConfigForTestNilConfig(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()
	err := c.ApplyConfigForTest(testCtxBg(), nil)
	require.NoError(t, err)
}

// TestApplyConfigForTestEmptyHTTPConfig exercises the sanitize+apply path with
// an empty HTTP config (no routers/middlewares/services).
func TestApplyConfigForTestEmptyHTTPConfig(t *testing.T) {
	t.Parallel()

	c := newTestController(t)
	cfg := &traefikconfig.Config{}
	err := c.ApplyConfigForTest(testCtxBg(), cfg)
	require.NoError(t, err)
}

// TestGraceQueueUnknownKindSkips verifies that unknown kinds are skipped
// without panicking.
func TestGraceQueueUnknownKindSkips(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		GCGraceQueueSize: 256,
		GCGracePeriod:    0,
		GCWorkers:        1,
	}
	collector := prometheus.NewCollector()
	c := NewController(cfg, nil, nil, collector)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.startGraceDeletionPool(ctx, 1)

	require.NotPanics(t, func() {
		c.enqueueGraceDeletion(ctx, graceDeleteReq{kind: "UnknownBogusKind", name: "unknown-item", delay: 0})
	})
}

func TestGraceQueueFallbackDoesNotBlockFullDelay(t *testing.T) {
	// This test forces the grace deletion queue to fill up while a worker is
	// blocked on a long grace delay, which triggers the synchronous fallback path
	// in enqueueGraceDeletion.

	listKinds := map[schema.GroupVersionResource]string{
		{Group: testutil.TestTraefikGroup, Version: "v1alpha1", Resource: "ingressroutes"}: "IngressRouteList",
	}
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds)
	kube := fakekube.NewClientset()
	cfg := &config.Config{
		Namespace:         "default",
		ManagedLabelKey:   "app.kubernetes.io/managed-by",
		ManagedLabelValue: "pangolin",
		GCGraceQueueSize:  1,
		GCWorkers:         1,
	}
	c := NewController(cfg, dyn, kube, nil)

	poolCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	longDelay := 10 * time.Second

	// First request: ensure a worker is busy waiting on delay.
	c.enqueueGraceDeletion(poolCtx, graceDeleteReq{kind: "IngressRoute", name: "one", delay: longDelay})
	require.Eventually(t, func() bool {
		q := c.graceDelQueue
		if q == nil {
			return false
		}
		return len(q) == 0
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Second request: fills the queue buffer.
	c.enqueueGraceDeletion(poolCtx, graceDeleteReq{kind: "IngressRoute", name: "two", delay: longDelay})
	require.Eventually(t, func() bool {
		q := c.graceDelQueue
		if q == nil {
			return false
		}
		return len(q) == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Third request: must hit the fallback path and should return in a bounded
	// time (queue wait ~500ms + capped fallback delay).
	start := time.Now()
	c.enqueueGraceDeletion(poolCtx, graceDeleteReq{kind: "IngressRoute", name: "three", delay: longDelay})
	elapsed := time.Since(start)

	require.Less(t, elapsed, 2*time.Second, "fallback enqueue should not block for the full grace delay")
}

func TestGraceQueueFallbackCancelsPromptly(t *testing.T) {
	// This test forces the grace deletion queue to be full (no worker draining),
	// then cancels the context while the synchronous fallback is waiting on its
	// bounded delay.

	listKinds := map[schema.GroupVersionResource]string{
		{Group: testutil.TestTraefikGroup, Version: "v1alpha1", Resource: "ingressroutes"}: "IngressRouteList",
	}
	gvr := schema.GroupVersionResource{Group: testutil.TestTraefikGroup, Version: "v1alpha1", Resource: "ingressroutes"}
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds)
	kube := fakekube.NewClientset()
	cfg := &config.Config{
		Namespace:         "default",
		ManagedLabelKey:   "app.kubernetes.io/managed-by",
		ManagedLabelValue: "pangolin",
		GCGraceQueueSize:  1,
		GCWorkers:         1,
	}
	c := NewController(cfg, dyn, kube, nil)

	// Make the queue exist but do not start workers, so the buffer stays full.
	c.graceDelQueue = make(chan graceDeleteReq, 1)
	c.graceDelQueue <- graceDeleteReq{kind: "IngressRoute", name: "buffered", delay: 10 * time.Second}

	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": testutil.TestTraefikGroup + "/v1alpha1",
		"kind":       "IngressRoute",
		"metadata": map[string]interface{}{
			"name":      "three",
			"namespace": "default",
		},
	}}
	_, err := dyn.Resource(gvr).Namespace("default").Create(context.Background(), obj, metav1.CreateOptions{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		c.enqueueGraceDeletion(ctx, graceDeleteReq{kind: "IngressRoute", name: "three", delay: 10 * time.Second})
		done <- time.Since(start)
	}()

	// Wait long enough for enqueueGraceDeletion to enter the fallback branch
	// (queue full wait ~500ms), then cancel while it's waiting on the bounded
	// delay.
	time.Sleep(600 * time.Millisecond)
	cancelAt := time.Now()
	cancel()

	select {
	case elapsed := <-done:
		require.Less(t, time.Since(cancelAt), 500*time.Millisecond, "expected fallback to return promptly after cancellation")
		require.Greater(t, elapsed, 500*time.Millisecond, "expected to have entered queue-full fallback path")
	case <-time.After(2 * time.Second):
		t.Fatal("enqueueGraceDeletion did not return promptly after cancellation")
	}

	// Ensure the synchronous delete was not executed after cancellation began.
	_, err = dyn.Resource(gvr).Namespace("default").Get(context.Background(), "three", metav1.GetOptions{})
	require.NoError(t, err)
}
