// Package orchestration provides the top-level orchestration logic for
// running the Pangolin Kubernetes Controller process. It wires together
// config loading, metrics/HTTP server startup, optional leader election
// and the reconciliation controller.
package orchestration

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/sync/errgroup"

	"pangolin-kube-controller/internal/config"
	"pangolin-kube-controller/internal/controller"
	inthttp "pangolin-kube-controller/internal/httpserver"
	"pangolin-kube-controller/internal/kube"
	"pangolin-kube-controller/internal/kube/labels"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
	"pangolin-kube-controller/internal/version"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// seams for testability in runController
var (
	newClients           = kube.NewClients
	resolveInstanceLabel = labels.ResolveInstanceLabel
	monitorInstanceLabel = labels.Monitor
)

// controllerFacade defines the minimal surface from the service.Controller
// used by runController. Introducing this interface enables injecting a
// lightweight fake in tests without changing public behavior.
type controllerFacade interface {
	Ready() bool
	RunLeaderElection(ctx context.Context)
	ExitRequested() bool
}

const metricsShutdownLog = "metrics server shutdown error: %v"

// ErrHTTPServerImmediateExit is returned when the HTTP server exits immediately
// without reporting an error. Tests can compare against this sentinel value
// using errors.Is to avoid fragile string comparisons.
var ErrHTTPServerImmediateExit = errors.New("http server exited immediately without error")

// makeController is a seam for constructing the controller. Defaults to
// controller.NewController but can be overridden in tests.
var makeController = func(cfg *config.Config, dyn dynamic.Interface, kube kubernetes.Interface, mc *prometheus.Collector) controllerFacade {
	return controller.NewController(cfg, dyn, kube, mc)
}

// startHTTPServer is a package-level test seam retained for backward compatibility.
// Deprecated: prefer passing an explicit HTTPStartFunc to RunWithStarter or the
// WithStarter helpers. Tests should avoid mutating this global; instead pass a
// start function to the WithStarter variants to avoid test flakiness from
// global state changes.
var startHTTPServer = func(httpSrv *inthttp.Server) error { return httpSrv.Start() }

// HTTPStartFunc is the function signature used to start the HTTP server. Tests
// may pass a custom implementation to RunWithStarter or the WithStarter
// helpers to avoid mutating package-level state.
type HTTPStartFunc func(httpSrv *inthttp.Server) error

// RunWithStarter is like Run but accepts a start function for the HTTP server
// so callers (tests) can inject deterministic behavior without modifying a
// package-global variable. Callers should pass nil to use the default
// startHTTPServer.
func RunWithStarter(ctx context.Context, cfg *config.Config, startFn HTTPStartFunc) error {
	if startFn == nil {
		startFn = startHTTPServer
	}
	logrus.Infof("Starting Pangolin Kubernetes Controller version=%s commit=%s endpoint=%s interval=%s namespace=%s leaderElection=%v", version.Version, version.Commit, cfg.Endpoint, cfg.PollInterval, cfg.Namespace, cfg.LeaderEnabled)

	metricsCollector := prometheus.NewCollector()
	httpSrv := inthttp.NewServer(cfg, metricsCollector.Handler())

	if cfg.StandaloneHTTPOnly {
		return runHTTPOnlyWithStarter(ctx, httpSrv, startFn)
	}
	return runControllerWithStarter(ctx, cfg, httpSrv, metricsCollector, startFn)
}

// Run boots the Pangolin controller application based on the supplied configuration.
// It returns when the context is cancelled or a fatal error occurs.
func Run(ctx context.Context, cfg *config.Config) error { //nolint:revive // orchestration entrypoint
	// Delegate to the parameterized variant using the package-default start
	// function to preserve historical behavior. Tests should use
	// RunWithStarter to inject a custom start function instead of mutating the
	// package-level seam.
	return RunWithStarter(ctx, cfg, nil)
}

// runHTTPOnly starts only the HTTP server and manages its lifecycle without running any reconciliation.
//
// runHTTPOnly marks the server as not ready, starts the provided HTTP server, and waits until either the
// provided context is canceled or the server exits. On exit or cancellation it attempts a graceful shutdown
// using the incoming context with a 5-second timeout. It returns any non-canceled error encountered while
// running or shutting down the server; if the server exited immediately without an error, it returns an
// explicit error indicating the premature exit.
func runHTTPOnly(ctx context.Context, httpSrv *inthttp.Server) error {
	// Delegate to the parameterized variant using the package default start function.
	return runHTTPOnlyWithStarter(ctx, httpSrv, startHTTPServer)
}

// runHTTPOnlyWithStarter is like runHTTPOnly but accepts an injected start
// function for the HTTP server. Tests should use this to avoid mutating the
// package-global seam.
func runHTTPOnlyWithStarter(ctx context.Context, httpSrv *inthttp.Server, startFn HTTPStartFunc) error {
	// In standalone HTTP mode we never become ready because no reconciliation occurs.
	httpSrv.SetReadinessFunc(func() bool { return false })
	g, gctx := errgroup.WithContext(ctx)
	srvExit := make(chan error, 1)
	g.Go(func() error {
		err := startFn(httpSrv)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvExit <- err
			return err
		}
		// Start returned (nil or server closed) – notify exit to avoid hangs when it returns immediately.
		srvExit <- err
		return nil
	})
	var srvExitErr error
	var serverExitedEarly bool
	select {
	case <-gctx.Done():
		// context cancelled or a goroutine returned error
	case err := <-srvExit:
		// HTTP server exited; store error to propagate after shutdown
		srvExitErr = err
		serverExitedEarly = true
	}
	// Derive from the provided ctx but strip cancellation to preserve a graceful shutdown window
	// even if the parent ctx was already cancelled.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logrus.Warnf(metricsShutdownLog, err)
	}
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	// If server exited early, check for unexpected nil error or propagate actual error
	if serverExitedEarly {
		if srvExitErr == nil {
			return ErrHTTPServerImmediateExit
		}
		return srvExitErr
	}
	return nil
}

// runController orchestrates the full controller lifecycle: it creates Kubernetes and dynamic clients,
// resolves the Traefik instance label, constructs the reconciliation controller, sets the HTTP readiness
// probe, runs the HTTP server, leader election, and instance-label monitoring in parallel, and coordinates
// graceful shutdown.
//
// It returns an error if client creation or instance-label resolution fails, if any background task
// terminates with a non-canceled error, or if leadership is lost and the controller has requested exit.
func runController(ctx context.Context, cfg *config.Config, httpSrv *inthttp.Server, metricsCollector *prometheus.Collector) error { //nolint:revive // orchestration helper
	// Delegate to the parameterized variant using the package default start function.
	return runControllerWithStarter(ctx, cfg, httpSrv, metricsCollector, startHTTPServer)
}

// runControllerWithStarter is like runController but accepts an injected start
// function for the HTTP server. Tests should call this variant when they need
// to control HTTP start behavior deterministically.
func runControllerWithStarter(ctx context.Context, cfg *config.Config, httpSrv *inthttp.Server, metricsCollector *prometheus.Collector, startFn HTTPStartFunc) error { //nolint:revive // orchestration helper
	clients, err := newClients(cfg)
	if err != nil {
		return err
	}

	// Resolve Traefik instance label (ENV/Config or autodetect via IngressClass) before starting reconciliation
	if err := resolveInstanceLabel(ctx, clients.Kubernetes, cfg, metricsCollector); err != nil {
		return err
	}

	ctrl := makeController(cfg, clients.Dynamic, clients.Kubernetes, metricsCollector)
	httpSrv.SetReadinessFunc(ctrl.Ready)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := startFn(httpSrv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		ctrl.RunLeaderElection(gctx)
		return nil
	})
	g.Go(func() error {
		return monitorInstanceLabel(gctx, clients.Kubernetes, cfg, metricsCollector)
	})

	<-gctx.Done()
	// Derive from the provided ctx but strip cancellation to preserve a graceful shutdown window
	// even if the parent ctx was already cancelled.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logrus.Warnf(metricsShutdownLog, err)
	}

	// Flush OpenTelemetry metrics before exit
	if metricsCollector.OTel != nil {
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer flushCancel()
		if mp, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); ok {
			if err := mp.Shutdown(flushCtx); err != nil {
				logrus.Warnf("MeterProvider shutdown: %v", err)
			}
		}
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	if ctrl.ExitRequested() {
		return errors.New("leadership lost: exit requested")
	}
	return nil
}
