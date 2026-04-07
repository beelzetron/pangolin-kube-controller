package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

func noop() {
	// Intentionally empty: used as a no-op callback in tests.
}

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
	cbs := c.leaderCallbacks("my-id", noop)

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

// TestPauseBehaviorSetsPausedFlag verifies that handleLeadershipLoss
// with pause behavior sets the paused flag.
func TestPauseBehaviorSetsPausedFlag(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "pause"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	require.False(t, c.paused.Load(), "paused should be false initially")
	c.handleLeadershipLoss(noop)
	require.True(t, c.paused.Load(), "paused should be true after pause behavior")
}

// TestExitBehaviorDoesNotSetPausedFlag verifies that handleLeadershipLoss
// with exit behavior does not set the paused flag.
func TestExitBehaviorDoesNotSetPausedFlag(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "exit"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	require.False(t, c.paused.Load(), "paused should be false initially")
	c.handleLeadershipLoss(noop)
	require.False(t, c.paused.Load(), "paused should still be false after exit behavior")
}

// TestRunCtxNotCancelledOnPause verifies that when ON_LOSE=pause,
// the runCtx is NOT cancelled (so the loop keeps running but is paused).
func TestRunCtxNotCancelledOnPause(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "pause"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	c.runCtx = context.Background()

	cancelled := false
	c.runCancel = func() { cancelled = true }

	c.handleLeadershipLoss(noop)

	require.False(t, cancelled, "runCtx should NOT be cancelled on pause")
	require.True(t, c.paused.Load(), "paused flag should be set")
}

// TestRunCtxCancelledOnExit verifies that when ON_LOSE=exit,
// the runCtx IS cancelled to stop the loop.
func TestRunCtxCancelledOnExit(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "exit"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	c.runCtx = context.Background()

	cancelled := false
	c.runCancel = func() { cancelled = true }

	c.handleLeadershipLoss(noop)

	require.True(t, cancelled, "runCtx should be cancelled on exit")
	require.True(t, c.ExitRequested(), "exitRequested should be set")
}

// TestExitCodeSetOnLeadershipLoss verifies that handleLeadershipLoss
// with exit behavior sets the exit code to ExitCodeLeadershipLoss.
func TestExitCodeSetOnLeadershipLoss(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "exit"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	require.Equal(t, int32(0), c.ExitCode(), "exit code should be 0 initially")
	c.handleLeadershipLoss(noop)
	require.Equal(t, int32(ExitCodeLeadershipLoss), c.ExitCode(), "exit code should be 2 after leadership loss with exit")
}

// TestExitCodeNotSetOnPause verifies that handleLeadershipLoss
// with pause behavior does not set the exit code.
func TestExitCodeNotSetOnPause(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{OnLoseBehavior: "pause"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())

	require.Equal(t, int32(0), c.ExitCode(), "exit code should be 0 initially")
	c.handleLeadershipLoss(noop)
	require.Equal(t, int32(0), c.ExitCode(), "exit code should still be 0 after pause")
}

// TestLastErrorStored verifies that LastError can store and retrieve errors.
func TestLastErrorStored(t *testing.T) {
	t.Parallel()

	c := newCtrlForTest()

	require.Nil(t, c.LastError(), "LastError should be nil initially")

	err := fmt.Errorf("test error")
	c.lastError.Store(err)

	require.Equal(t, err, c.LastError(), "LastError should return stored error")
}
