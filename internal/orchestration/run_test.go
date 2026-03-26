package orchestration

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"pangolin-kube-controller/internal/config"
	inthttp "pangolin-kube-controller/internal/httpserver"
	"pangolin-kube-controller/internal/kube"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const unexpectedErrFmt = "unexpected error: %v"

// sentinel error used by tests to make error comparisons robust
var errMonitorFailed = errors.New("monitor failed")

// TestRunHTTPOnlyMode mirrors the app package test for the controller facade.
func TestRunHTTPOnlyMode(t *testing.T) {
	// Reserve a listener to avoid races when checking port availability.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: fmt.Sprintf(":%d", port)}

	// Create the HTTP server and start it on our reserved listener. This
	// ensures the test controls the exact net.Listener used by the server and
	// avoids races where another process could bind the port between closing
	// the listener and starting the server.
	metricsCollector := prometheus.NewCollector()
	srv := inthttp.NewServer(cfg, metricsCollector.Handler())
	// In HTTP-only mode readiness is always false
	srv.SetReadinessFunc(func() bool { return false })

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(ln)
	}()

	// Poll the /healthz endpoint until it becomes reachable (200 or 503) indicating server started.
	deadline := time.Now().Add(2 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	for {
		if time.Now().After(deadline) {
			// Attempt a graceful shutdown to free the listener, then fail the test.
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			_ = srv.Shutdown(shutdownCtx)
			shutdownCancel()
			t.Fatalf("server failed to start within timeout")
		}
		client := &http.Client{Timeout: 150 * time.Millisecond}
		resp, err := client.Get(url)
		if err == nil {
			// In HTTP-only mode readiness is always false, so 503 is acceptable; 200 would also be fine if logic changes.
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("unexpected status code %d", resp.StatusCode)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Now that server is confirmed listening, trigger shutdown.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("server shutdown error: %v", err)
	}
	select {
	case err := <-serveErrCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("unexpected serve error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not stop in time")
	}
	// Close the reserved listener after the server has stopped to avoid races
	// between Serve and closing the listener.
	_ = ln.Close()
}

// Tests from run_more_test.go

func TestRunControllerNewClientsError(t *testing.T) {
	// Disable pprof and provide a metrics addr so HTTP server initializes without plaintext
	// and runController proceeds to NewClients where the test expects an error.
	cfg := &config.Config{EnablePprof: false, MetricsPlaintextOK: false, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()
	ctx := context.Background()
	if runController(ctx, cfg, httpSrv, mc) == nil {
		t.Fatalf("expected error from NewClients in controller.runController")
	}
}

func TestRunNonStandaloneReturnsError(t *testing.T) {
	cfg := &config.Config{} // default non-standalone
	ctx := context.Background()
	if Run(ctx, cfg) == nil {
		t.Fatalf("expected error from controller.Run when not in standalone mode (no cluster)")
	}
}

func TestRunControllerResolveInstanceLabelError(t *testing.T) {
	// Stub newClients to return a minimal client set without requiring cluster
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return errors.New("fail")
	}
	t.Cleanup(func() { newClients = oldNC; resolveInstanceLabel = oldResolve })

	cfg := &config.Config{MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if runController(ctx, cfg, httpSrv, mc) == nil {
		t.Fatalf("expected error when ResolveInstanceLabel fails")
	}
}

func TestRunHTTPOnlyFastCancel(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := runHTTPOnly(ctx, httpSrv); err != nil {
		t.Fatalf(unexpectedErrFmt, err)
	}
}

func TestRunHTTPOnlyMisconfigFailsFast(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, EnablePprof: true, MetricsPlaintextOK: false, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if runHTTPOnly(ctx, httpSrv) == nil {
		t.Fatalf("expected error when HTTP server exits immediately due to misconfiguration")
	}
}

// New coverage: exercise Run with StandaloneHTTPOnly flow and both success and misconfig error paths.
func TestRunStandaloneHTTPOnlyCancel(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	// Use injected start function to deterministically know when the HTTP server has started.
	startedCh := make(chan struct{}, 1)
	startFn := func(s *inthttp.Server) error {
		// Signal that Start has been invoked, then call the real implementation.
		select {
		case startedCh <- struct{}{}:
		default:
		}
		return startHTTPServer(s)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- RunWithStarter(ctx, cfg, startFn) }()

	// Wait until the server start seam is invoked, then cancel.
	select {
	case <-startedCh:
		cancel()
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatalf("server did not start within timeout")
	}

	// Wait for Run to exit and assert acceptable result.
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf(unexpectedErrFmt, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not exit after cancellation")
	}
}

func TestRunStandaloneHTTPOnlyMisconfigFailsFast(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, EnablePprof: true, MetricsPlaintextOK: false, MetricsAddr: ":0"}
	// Run should surface the same misconfiguration error as runHTTPOnly
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if Run(ctx, cfg) == nil {
		t.Fatalf("expected error from Run in standalone HTTP-only misconfiguration")
	}
}

// Tests for context propagation in shutdown

func TestRunHTTPOnlyShutdownUsesParentContext(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)

	// Create a context that will be cancelled to trigger shutdown
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHTTPOnly(ctx, httpSrv)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel parent context to trigger shutdown
	cancel()

	// Wait for shutdown to complete
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf(unexpectedErrFmt, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for shutdown")
	}
}

func TestRunHTTPOnlyShutdownRespectsParentCancellation(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// runHTTPOnly should return quickly since context is already cancelled
	start := time.Now()
	err := runHTTPOnly(ctx, httpSrv)
	elapsed := time.Since(start)

	// Should complete quickly (within 1 second) since context was pre-cancelled
	if elapsed > time.Second {
		t.Fatalf("shutdown took too long with pre-cancelled context: %v", elapsed)
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf(unexpectedErrFmt, err)
	}
}

func TestRunControllerShutdownUsesParentContext(t *testing.T) {
	// Stub newClients to return a minimal client set
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return nil
	}
	t.Cleanup(func() { newClients = oldNC; resolveInstanceLabel = oldResolve })

	cfg := &config.Config{MetricsAddr: ":0", PollInterval: 10 * time.Millisecond}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- runController(ctx, cfg, httpSrv, mc)
	}()

	// Give controller time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel parent context
	cancel()

	// Wait for shutdown
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf(unexpectedErrFmt, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for controller shutdown")
	}
}

func TestRunControllerShutdownWithAlreadyCancelledContext(t *testing.T) {
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return nil
	}
	t.Cleanup(func() { newClients = oldNC; resolveInstanceLabel = oldResolve })

	cfg := &config.Config{MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()

	// Create pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := runController(ctx, cfg, httpSrv, mc); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf(unexpectedErrFmt, err)
	}
	elapsed := time.Since(start)
	// Should complete quickly
	if elapsed > 2*time.Second {
		t.Fatalf("shutdown took too long with pre-cancelled context: %v", elapsed)
	}
}

// New coverage: ensure runController returns an explicit error when the controller
// has requested exit (leadership lost). Uses injected seams to avoid real clients.
func TestRunControllerExitRequestedError(t *testing.T) {
	// Save and restore seams
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	oldMonitor := monitorInstanceLabel
	oldMake := makeController

	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return nil
	}
	monitorInstanceLabel = func(ctx context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		// Return when context is done to allow clean shutdown
		<-ctx.Done()
		return nil
	}
	makeController = func(_ *config.Config, _ dynamic.Interface, _ kubernetes.Interface, _ *prometheus.Collector) controllerFacade {
		return &fakeController{ready: true, exitRequested: true}
	}
	t.Cleanup(func() {
		newClients = oldNC
		resolveInstanceLabel = oldResolve
		monitorInstanceLabel = oldMonitor
		makeController = oldMake
	})

	cfg := &config.Config{MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := runController(ctx, cfg, httpSrv, mc)
	if err == nil || err.Error() != "leadership lost: exit requested" {
		t.Fatalf("expected exit-requested error, got: %v", err)
	}
}

// fakeController is a lightweight stub implementing the minimal facade used by runController.
type fakeController struct {
	ready         bool
	exitRequested bool
}

func (f *fakeController) Ready() bool                         { return f.ready }
func (*fakeController) RunLeaderElection(ctx context.Context) { <-ctx.Done() }
func (f *fakeController) ExitRequested() bool                 { return f.exitRequested }

// New coverage: runController should return the HTTP server Start error when misconfigured
func TestRunControllerMetricsPlaintextMisconfig(t *testing.T) {
	// Stub seams to allow runController to proceed to HTTP server startup
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return nil
	}
	t.Cleanup(func() { newClients = oldNC; resolveInstanceLabel = oldResolve })

	cfg := &config.Config{EnablePprof: true, MetricsPlaintextOK: false, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if runController(ctx, cfg, httpSrv, mc) == nil {
		t.Fatalf("expected error from HTTP server Start misconfiguration")
	}
}

// New coverage: runController should return monitor error when instance label monitor fails
func TestRunControllerMonitorError(t *testing.T) {
	oldNC := newClients
	oldResolve := resolveInstanceLabel
	oldMonitor := monitorInstanceLabel
	newClients = func(_ *config.Config) (*kube.Clients, error) { return &kube.Clients{}, nil }
	resolveInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return nil
	}
	monitorInstanceLabel = func(_ context.Context, _ kubernetes.Interface, _ *config.Config, _ *prometheus.Collector) error {
		return errMonitorFailed
	}
	t.Cleanup(func() { newClients = oldNC; resolveInstanceLabel = oldResolve; monitorInstanceLabel = oldMonitor })

	cfg := &config.Config{MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)
	mc := prometheus.NewCollector()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runController(ctx, cfg, httpSrv, mc)
	if err == nil || !errors.Is(err, errMonitorFailed) { // returned directly from monitorInstanceLabel
		t.Fatalf("expected monitor failure error, got: %v", err)
	}
}

// New coverage: simulate immediate server exit with nil error via seam to exercise
// the branch returning an explicit error and the shutdown warning path.
func TestRunHTTPOnlyImmediateNilExit(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)

	// Use injected start function to simulate immediate nil exit.
	startFn := func(_ *inthttp.Server) error { return nil }

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := runHTTPOnlyWithStarter(ctx, httpSrv, startFn)
	if err == nil || !errors.Is(err, ErrHTTPServerImmediateExit) {
		t.Fatalf("expected explicit immediate-exit sentinel error, got: %v", err)
	}
}

func TestRunHTTPOnlyImmediateServerClosed(t *testing.T) {
	cfg := &config.Config{StandaloneHTTPOnly: true, MetricsAddr: ":0"}
	httpSrv := inthttp.NewServer(cfg, nil)

	startFn := func(_ *inthttp.Server) error { return http.ErrServerClosed }

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := runHTTPOnlyWithStarter(ctx, httpSrv, startFn)
	if !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("expected ErrServerClosed, got: %v", err)
	}
}
