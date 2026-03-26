package metrics_prometheus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShutdownOTelNilOTel confirms that ShutdownOTel returns nil when OTel
// is not initialised (nil pointer) – the nil guard must be exercised.
func TestShutdownOTelNilOTel(t *testing.T) {
	t.Parallel()

	c := &Collector{} // OTel is nil
	err := c.ShutdownOTel(context.Background())
	require.NoError(t, err, "ShutdownOTel with nil OTel must return nil")
}

// TestShutdownOTelWithRealCollector confirms that calling ShutdownOTel on a
// fully-initialised Collector (which sets up an OTel MeterProvider) does not
// error. This exercises the non-nil OTel.Shutdown path.
func TestShutdownOTelWithRealCollector(t *testing.T) {
	t.Parallel()

	c := NewCollector()
	if c.OTel == nil {
		t.Skip("OTel not initialised in this environment; skipping ShutdownOTel with OTel test")
	}
	err := c.ShutdownOTel(context.Background())
	require.NoError(t, err, "ShutdownOTel must succeed when OTel is initialised")
}
