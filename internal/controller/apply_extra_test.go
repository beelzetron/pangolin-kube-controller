package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"pangolin-kube-controller/internal/config"
	traefikconfig "pangolin-kube-controller/internal/transform/config"
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
