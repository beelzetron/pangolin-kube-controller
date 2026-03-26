package controller

import (
	"context"
	"testing"
	"time"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

func TestHandleLeadershipLossExitSetsFlagAndCancels(t *testing.T) {
	cfg := &config.Config{OnLoseBehavior: "exit"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	cancelled := false
	cancel := func() { cancelled = true }
	c.handleLeadershipLoss(cancel)
	if !c.ExitRequested() {
		t.Fatalf("expected exit requested flag set")
	}
	if !cancelled {
		t.Fatalf("expected cancel to be called")
	}
}

func TestHandleLeadershipLossPauseDoesNotExit(t *testing.T) {
	cfg := &config.Config{OnLoseBehavior: "pause"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	cancelled := false
	cancel := func() { cancelled = true }
	c.handleLeadershipLoss(cancel)
	if c.ExitRequested() {
		t.Fatalf("did not expect exit requested for pause")
	}
	if cancelled {
		t.Fatalf("did not expect cancel for pause")
	}
}

func TestHandleLeadershipLossUnknownTreatsAsExit(t *testing.T) {
	cfg := &config.Config{OnLoseBehavior: "other"}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	cancelled := false
	cancel := func() { cancelled = true }
	c.handleLeadershipLoss(cancel)
	if !c.ExitRequested() {
		t.Fatalf("expected exit requested for unknown behavior")
	}
	if !cancelled {
		t.Fatalf("expected cancel to be called for unknown behavior")
	}
}

func TestOnStartedLeadingRespectsContext(t *testing.T) {
	c := NewController(&config.Config{}, nil, nil, prometheus.NewCollector())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	panicCh := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// send recovered panic back to main goroutine
				panicCh <- r
			} else {
				panicCh <- nil
			}
			close(done)
		}()
		c.OnStartedLeading(ctx, "id-x")
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("OnStartedLeading did not return within timeout")
	}
	// check for panic captured from goroutine and report in test goroutine
	if rec := <-panicCh; rec != nil {
		t.Fatalf("OnStartedLeading panicked: %v", rec)
	}
}
