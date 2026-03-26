package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeBackoffDurationBounds(t *testing.T) {
	c := newCtrlForTest()
	c.cfg.PollInterval = 100 * time.Millisecond
	c.cfg.MaxBackoff = 2 * time.Second

	d0 := c.computeBackoffDuration(0)
	require.Equal(t, c.cfg.PollInterval, d0)

	d1 := c.computeBackoffDuration(1)
	require.GreaterOrEqual(t, d1, time.Duration(float64(c.cfg.PollInterval)*0.8))
	require.LessOrEqual(t, d1, time.Duration(float64(c.cfg.PollInterval)*1.2))

	d5 := c.computeBackoffDuration(5)
	require.LessOrEqual(t, d5, c.cfg.MaxBackoff)
}
