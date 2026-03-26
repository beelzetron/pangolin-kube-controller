package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

// TestOnStoppedLeadingCallsHandleLeadershipLoss confirms that OnStoppedLeading
// delegates correctly to handleLeadershipLoss for exit behavior.
func TestOnStoppedLeadingExitBehavior(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "exit"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	cancelled := false
	cancel := func() { cancelled = true }
	c.OnStoppedLeading(cancel)

	require.True(t, c.ExitRequested(), "expected exit requested after OnStoppedLeading")
	require.True(t, cancelled, "expected cancel() to be called")
}

// TestOnStoppedLeadingPauseBehavior confirms no exit is requested when using
// the "pause" on-lose behavior.
func TestOnStoppedLeadingPauseBehavior(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "pause"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	cancelled := false
	cancel := func() { cancelled = true }
	c.OnStoppedLeading(cancel)

	require.False(t, c.ExitRequested(), "pause behavior must not set exit")
	require.False(t, cancelled, "pause behavior must not call cancel()")
}

// TestOnNewLeaderSameID checks that onNewLeader does not panic when the
// current leader matches the local identity (no log emitted).
func TestOnNewLeaderSameID(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		onNewLeader("leader-abc", "leader-abc")
	})
}

// TestOnNewLeaderDifferentID checks that a new different leader is handled
// without panic.
func TestOnNewLeaderDifferentID(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		onNewLeader("old-leader", "new-leader")
	})
}

// TestLeaderCallbacksOnNewLeaderInvocation verifies that the OnNewLeader
// callback returned by leaderCallbacks calls onNewLeader without panicking.
func TestLeaderCallbacksOnNewLeaderInvocation(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()
	cbs := c.leaderCallbacks("my-id", func() {})

	require.NotPanics(t, func() {
		cbs.OnNewLeader("my-id")    // same id – no log
		cbs.OnNewLeader("other-id") // different id – logs
	})
}

// TestLeaderCallbacksOnStoppedLeading verifies the OnStoppedLeading callback
// delegates to handleLeadershipLoss.
func TestLeaderCallbacksOnStoppedLeading(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()
	c.cfg.OnLoseBehavior = "exit"
	cancelled := false
	cancel := func() { cancelled = true }

	cbs := c.leaderCallbacks("my-id", cancel)
	cbs.OnStoppedLeading()

	require.True(t, c.ExitRequested())
	require.True(t, cancelled)
}

// TestOnStartedLeadingCancelledContext ensures OnStartedLeading returns
// quickly when the context is already cancelled.
func TestOnStartedLeadingAlreadyCancelled(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	require.NotPanics(t, func() {
		c.OnStartedLeading(ctx, "node-xyz")
	})
}
