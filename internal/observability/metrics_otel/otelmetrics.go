package metrics_otel

import (
	"context"
	"errors"
	"os"
	"time"

	stdprom "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"pangolin-kube-controller/internal/version"
)

var defaultDurationBuckets = []float64{
	0.001, // 1ms
	0.005, // 5ms
	0.01,  // 10ms
	0.025, // 25ms
	0.05,  // 50ms
	0.1,   // 100ms
	0.25,  // 250ms
	0.5,   // 500ms
	1.0,   // 1s
	2.5,   // 2.5s
	5.0,   // 5s
	10.0,  // 10s
}

// OTel holds OpenTelemetry metric instruments used by the controller.
type OTel struct {
	MeterProvider *sdkmetric.MeterProvider
	Meter         metric.Meter

	ReconcilePhaseDuration  metric.Float64Histogram
	FetchDuration           metric.Float64Histogram
	K8sRequestDuration      metric.Float64Histogram
	K8sRequestsTotal        metric.Int64Counter
	RetriesTotal            metric.Int64Counter
	ActiveReconcileRoutines metric.Int64UpDownCounter
	GCRunDuration           metric.Float64Histogram
	ConfigParseDuration     metric.Float64Histogram
	LoopIterationsTotal     metric.Int64Counter
}

// Testability seams (default to real constructors). These vars allow unit tests to
// simulate constructor failures without changing production behavior.
var (
	newPromExporter = func(reg stdprom.Registerer) (*otelprom.Exporter, error) {
		return otelprom.New(otelprom.WithRegisterer(reg))
	}
	newMeterProviderFromReader = func(reader sdkmetric.Reader, res *resource.Resource) *sdkmetric.MeterProvider {
		return sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(res),
		)
	}
	makeF64HistFn = func(meter metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
		return meter.Float64Histogram(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
			metric.WithExplicitBucketBoundaries(defaultDurationBuckets...),
		)
	}
	makeI64CtrFn = func(meter metric.Meter, name, desc string) (metric.Int64Counter, error) {
		return meter.Int64Counter(name, metric.WithDescription(desc))
	}
	makeI64UpDownFn = func(meter metric.Meter, name, desc string) (metric.Int64UpDownCounter, error) {
		return meter.Int64UpDownCounter(name, metric.WithDescription(desc))
	}
)

// SetupOTel wires an OpenTelemetry MeterProvider and Prometheus exporter onto the provided registry.
// The returned instruments are recorded alongside existing Prometheus metrics at the same /metrics endpoint.
func SetupOTel(reg *stdprom.Registry) (*OTel, error) {
	if reg == nil {
		return nil, errors.New("registry is nil")
	}
	setupCtx := context.Background()

	// Create Resource attributes for service identity
	res, err := resource.New(setupCtx,
		resource.WithAttributes(
			semconv.ServiceName("pangolin-kube-controller"),
			semconv.ServiceVersion(version.Version),
			semconv.ServiceInstanceID(getInstanceID()),
		),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := newPromExporter(reg)
	if err != nil {
		return nil, err
	}

	// Create MeterProvider with Resource attributes using the seam
	mp := newMeterProviderFromReader(exporter, res)
	meter := mp.Meter("pangolin-controller")

	// reuse err from earlier declarations
	ot := &OTel{Meter: meter}

	if ot.ReconcilePhaseDuration, err = makeF64HistFn(meter, "pangolin_controller_reconcile_phase_duration_seconds", "Duration of each reconcile phase.", "s"); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.FetchDuration, err = makeF64HistFn(meter, "pangolin_controller_fetch_duration_seconds", "Duration of remote fetch cycle HTTP requests.", "s"); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.K8sRequestDuration, err = makeF64HistFn(meter, "pangolin_controller_k8s_request_duration_seconds", "Duration of Kubernetes API requests.", "s"); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.K8sRequestsTotal, err = makeI64CtrFn(meter, "pangolin_controller_k8s_requests_total", "Total Kubernetes API requests."); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.RetriesTotal, err = makeI64CtrFn(meter, "pangolin_controller_retries_total", "Total number of retry attempts by reason."); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.ActiveReconcileRoutines, err = makeI64UpDownFn(meter, "pangolin_controller_active_reconcile_routines", "Number of active reconcile routines by phase."); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.GCRunDuration, err = makeF64HistFn(meter, "pangolin_controller_gc_run_duration_seconds", "Duration of GC runs.", "s"); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.ConfigParseDuration, err = makeF64HistFn(meter, "pangolin_controller_config_parse_duration_seconds", "Duration of config parsing by section.", "s"); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}
	if ot.LoopIterationsTotal, err = makeI64CtrFn(meter, "pangolin_controller_loop_iterations_total", "Number of controller loop iterations by outcome."); err != nil {
		_ = mp.Shutdown(setupCtx)
		return nil, err
	}

	// Only install the global provider after instrument creation succeeds.
	otel.SetMeterProvider(mp)

	// Start Go runtime metrics instrumentation
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		log.Warnf("Failed to start Go runtime metrics collection: %v. Runtime metrics (goroutines, memory, GC) will not be available.", err)
	}

	ot.MeterProvider = mp

	return ot, nil
}

// Shutdown flushes and shuts down the MeterProvider gracefully.
func (ot *OTel) Shutdown(ctx context.Context) error {
	if ot.MeterProvider == nil {
		return nil
	}
	return ot.MeterProvider.Shutdown(ctx)
}

// getInstanceID returns a unique identifier for this controller instance.
// It prefers POD_UID in Kubernetes, falls back to hostname, or returns "unknown".
func getInstanceID() string {
	if uid := os.Getenv("POD_UID"); uid != "" {
		return uid
	}
	if hn, err := os.Hostname(); err == nil {
		return hn
	}
	return "unknown"
}
