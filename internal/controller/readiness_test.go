package controller

import (
	"context"
	"testing"
	"time"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

func TestReadyTransitions(t *testing.T) {
	c := NewController(&config.Config{}, nil, nil, prometheus.NewCollector())
	c.lastSuccessfulReconcile.Store(0)
	if c.Ready() {
		t.Fatalf("expected not ready before any reconcile")
	}
	c.lastSuccessfulReconcile.Store(time.Now().UnixNano())
	if !c.Ready() {
		t.Fatalf("expected ready after recent reconcile")
	}
	c.cfg.PollInterval = 10 * time.Millisecond
	past := time.Now().Add(-5*c.cfg.PollInterval - time.Millisecond)
	c.lastSuccessfulReconcile.Store(past.UnixNano())
	if c.Ready() {
		t.Fatalf("expected not ready when last reconcile is stale")
	}
}

func TestReadyStateTransitions(t *testing.T) {
	cfg := &config.Config{PollInterval: 50 * time.Millisecond}
	c := NewController(cfg, nil, nil, prometheus.NewCollector())
	if c.Ready() {
		t.Fatalf("expected not ready initially")
	}
	c.lastSuccessfulReconcile.Store(time.Now().UnixNano())
	if !c.Ready() {
		t.Fatalf("expected ready after success")
	}
	old := time.Now().Add(-6 * cfg.PollInterval).UnixNano()
	c.lastSuccessfulReconcile.Store(old)
	if c.Ready() {
		t.Fatalf("expected not ready when last success too old")
	}
}

func TestStandalone(t *testing.T) {
	done := make(chan struct{})
	go func() {
		ctrl := &Controller{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ctrl.Standalone(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Standalone did not return")
	}
}
