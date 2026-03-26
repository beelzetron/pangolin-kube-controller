package metrics_otel

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// TestOTelShutdownNilMeterProvider verifies that calling Shutdown on an OTel
// whose MeterProvider is nil returns nil without panicking.
func TestOTelShutdownNilMeterProvider(t *testing.T) {
	t.Parallel()

	ot := &OTel{} // MeterProvider is nil
	err := ot.Shutdown(context.Background())
	require.NoError(t, err, "Shutdown with nil MeterProvider must return nil")
}

// TestOTelShutdownWithRealProvider verifies that Shutdown on an OTel with a
// real MeterProvider flushes and shuts down without error.
func TestOTelShutdownWithRealProvider(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	ot, err := SetupOTel(reg)
	if err != nil {
		t.Skipf("OTel not available in this environment: %v", err)
	}
	require.NotNil(t, ot)

	err = ot.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestOTelShutdownBareProvider exercises the Shutdown method with a freshly
// created (no-op) MeterProvider to confirm the path does not panic.
func TestOTelShutdownBareProvider(t *testing.T) {
	t.Parallel()

	mp := sdkmetric.NewMeterProvider()
	ot := &OTel{MeterProvider: mp}

	err := ot.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestGetInstanceIDPODUID verifies that when the POD_UID env var is set, its
// value is returned directly.
func TestGetInstanceIDPODUID(t *testing.T) {
	t.Setenv("POD_UID", "test-pod-uid-12345")
	id := getInstanceID()
	require.Equal(t, "test-pod-uid-12345", id)
}

// TestGetInstanceIDFallbackToHostname verifies that when POD_UID is unset, the
// function returns a non-empty string (either the real hostname or "unknown").
func TestGetInstanceIDFallbackToHostname(t *testing.T) {
	t.Setenv("POD_UID", "") // clear if set in environment
	id := getInstanceID()
	require.NotEmpty(t, id, "getInstanceID must not return empty string")
}
