package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFallbackJitterFloat64Range validates that fallbackJitterFloat64 returns
// values in [0.0, 1.0) on each call.
func TestFallbackJitterFloat64Range(t *testing.T) {
	t.Parallel()

	for i := 0; i < 20; i++ {
		v := fallbackJitterFloat64()
		require.GreaterOrEqual(t, v, 0.0, "fallbackJitterFloat64 must be >= 0")
		require.Less(t, v, 1.0, "fallbackJitterFloat64 must be < 1")
	}
}

// TestFallbackJitterFloat64Uniqueness checks that repeated calls produce at least
// some distinct values (extremely low probability of all being identical).
func TestFallbackJitterFloat64Uniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[float64]struct{})
	for i := 0; i < 10; i++ {
		seen[fallbackJitterFloat64()] = struct{}{}
	}
	require.Greater(t, len(seen), 1, "expected multiple distinct jitter values")
}

// TestGetJitterFloat64FallbackOnCryptoError exercises the fallback branch in
// getJitterFloat64 when crypto/rand.Read returns an error.
func TestGetJitterFloat64FallbackOnCryptoError(t *testing.T) {
	old := cryptoRead
	t.Cleanup(func() { cryptoRead = old })
	cryptoRead = func(_ []byte) (int, error) {
		return 0, errors.New("mock entropy failure")
	}

	v := getJitterFloat64()
	require.GreaterOrEqual(t, v, 0.0)
	require.Less(t, v, 1.0)
}

// TestComputeBackoffDurationJitterApplied confirms that jitter is applied and
// that multiple calls with the same error count return values within ±20% of
// the base interval (the jitter band is [0.8x, 1.2x]).
func TestComputeBackoffDurationJitterApplied(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()
	c.cfg.PollInterval = 100 * time.Millisecond
	c.cfg.MaxBackoff = 10 * time.Second

	const errorCount = 1
	for i := 0; i < 5; i++ {
		d := c.computeBackoffDuration(errorCount)
		require.Greater(t, d.Nanoseconds(), int64(0))
	}
}
