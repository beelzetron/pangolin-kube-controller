package metrics_otel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestOTelExporterRegistersToRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	ot, err := SetupOTel(reg)
	if err != nil {
		// If exporter cannot initialize in this env, skip (the controller logs a warning and continues).
		t.Skipf("otel exporter not available: %v", err)
	}
	// Emit some sample measurements.
	ctx := context.Background()
	ot.LoopIterationsTotal.Add(ctx, 1, metric.WithAttributes(AttrOutcome.String("success")))
	ot.FetchDuration.Record(ctx, 0.01,
		metric.WithAttributes(
			AttrStatusClass.String("2xx"),
		),
	)

	rr := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := rr.Body.String()
	require.Contains(t, body, "pangolin_controller_loop_iterations_total")
	require.Contains(t, body, "pangolin_controller_fetch_duration_seconds_bucket")
}

func TestStatusClass(t *testing.T) {
	cases := map[int]string{
		0:   "0",
		199: "1xx",
		200: "2xx",
		399: "3xx",
		400: "4xx",
		599: "5xx",
		-1:  "unknown",
		600: "unknown",
		42:  "unknown",
	}
	for code, want := range cases {
		got := StatusClass(code)
		if got != want {
			t.Fatalf("code %d: want %s, got %s", code, want, got)
		}
	}
}

func TestSetupOTelNilRegistry(t *testing.T) {
	ot, err := SetupOTel(nil)
	if err == nil || ot != nil {
		t.Fatalf("expected error for nil registry, got ot=%v err=%v", ot, err)
	}
}

// --- merged from otelmetrics_extra_test.go ---

// restore helpers
func withSeams(t *testing.T, fn func()) {
	oldNewExporter := newPromExporter
	oldNewMP := newMeterProviderFromReader
	oldMakeF64 := makeF64HistFn
	oldMakeI64Ctr := makeI64CtrFn
	oldMakeI64UpDown := makeI64UpDownFn
	t.Cleanup(func() {
		newPromExporter = oldNewExporter
		newMeterProviderFromReader = oldNewMP
		makeF64HistFn = oldMakeF64
		makeI64CtrFn = oldMakeI64Ctr
		makeI64UpDownFn = oldMakeI64UpDown
	})
	fn()
}

// successfulCtrAndUpDown sets counter and updown factories to succeed.
func successfulCtrAndUpDown() {
	makeI64CtrFn = func(m metric.Meter, name, desc string) (metric.Int64Counter, error) {
		return m.Int64Counter(name, metric.WithDescription(desc))
	}
	makeI64UpDownFn = func(m metric.Meter, name, desc string) (metric.Int64UpDownCounter, error) {
		return m.Int64UpDownCounter(name, metric.WithDescription(desc))
	}
}

// successfulHistAndUpDown sets histogram and updown factories to succeed.
func successfulHistAndUpDown() {
	makeF64HistFn = func(m metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
		return m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
	}
	makeI64UpDownFn = func(m metric.Meter, name, desc string) (metric.Int64UpDownCounter, error) {
		return m.Int64UpDownCounter(name, metric.WithDescription(desc))
	}
}

func TestSetupOTelExporterError(t *testing.T) {
	withSeams(t, func() {
		// Force exporter creation to fail
		newPromExporter = func(_ prometheus.Registerer) (*otelprom.Exporter, error) { return nil, errors.New("exporter failure") }
		reg := prometheus.NewRegistry()
		ot, err := SetupOTel(reg)
		if err == nil || ot != nil {
			t.Fatalf("expected non-nil error and nil otel on exporter failure, got ot=%v err=%v", ot, err)
		}
	})
}

func TestSetupOTelHistogramCreationError(t *testing.T) {
	withSeams(t, func() {
		// Ensure counters and updown factories succeed so histogram failure is the root cause.
		successfulCtrAndUpDown()
		makeF64HistFn = func(_ metric.Meter, _, _, _ string) (metric.Float64Histogram, error) {
			return nil, errors.New("histogram create failure")
		}
		reg := prometheus.NewRegistry()
		ot, err := SetupOTel(reg)
		if err == nil || ot != nil {
			t.Fatalf("expected non-nil error and nil otel from histogram creation failure, got ot=%v err=%v", ot, err)
		}
	})
}

func TestSetupOTelCounterCreationError(t *testing.T) {
	withSeams(t, func() {
		// Explicitly ensure histograms succeed so counter failure is exercised.
		makeF64HistFn = func(m metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
			// explicitly create a valid histogram to ensure this path succeeds
			return m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
		}
		makeI64CtrFn = func(_ metric.Meter, _, _ string) (metric.Int64Counter, error) {
			return nil, errors.New("counter create failure")
		}
		// Ensure UpDown counter creation also succeeds so this test deterministically
		// exercises the counter creation failure rather than failing earlier due
		// to instrument creation order differences.
		makeI64UpDownFn = func(m metric.Meter, name, desc string) (metric.Int64UpDownCounter, error) {
			return m.Int64UpDownCounter(name, metric.WithDescription(desc))
		}
		reg := prometheus.NewRegistry()
		ot, err := SetupOTel(reg)
		if err == nil || ot != nil {
			t.Fatalf("expected non-nil error and nil otel from counter creation failure, got ot=%v err=%v", ot, err)
		}
	})
}

func TestSetupOTelUpDownCreationError(t *testing.T) {
	withSeams(t, func() {
		// Ensure preceding factories succeed, then fail the updown counter.
		makeF64HistFn = func(m metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
			return m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
		}
		makeI64CtrFn = func(m metric.Meter, name, desc string) (metric.Int64Counter, error) {
			return m.Int64Counter(name, metric.WithDescription(desc))
		}
		makeI64UpDownFn = func(_ metric.Meter, _, _ string) (metric.Int64UpDownCounter, error) {
			return nil, errors.New("updown create failure")
		}
		reg := prometheus.NewRegistry()
		ot, err := SetupOTel(reg)
		if err == nil || ot != nil {
			t.Fatalf("expected non-nil error and nil otel from updown creation failure, got ot=%v err=%v", ot, err)
		}
	})
}

func TestSetupOTelSuccessWithFactoryOverrides(t *testing.T) {
	withSeams(t, func() {
		// Make exporter creation a no-op and provide a minimal meter provider without a reader
		// to ensure instrument construction succeeds deterministically when overriding factories.
		newPromExporter = func(_ prometheus.Registerer) (*otelprom.Exporter, error) { return nil, nil }
		newMeterProviderFromReader = func(_ sdkmetric.Reader, _ *resource.Resource) *sdkmetric.MeterProvider {
			return sdkmetric.NewMeterProvider()
		}
		reg := prometheus.NewRegistry()
		ot, err := SetupOTel(reg)
		if err != nil || ot == nil {
			t.Fatalf("expected success SetupOTel, got ot=%v err=%v", ot, err)
		}
		// Ensure the returned OTel contains a usable Meter (non-nil) and that creating
		// and using an instrument does not panic or return an error. We perform a
		// minimal instrument creation and a no-op record to validate end-to-end.
		if ot.Meter == nil {
			t.Fatalf("expected non-nil Meter on returned OTel")
		}
		// Attempt to create a temporary counter instrument and record a value.
		// Use the Meter interface directly; some SDKs defer errors until record time,
		// so ensure that recording does not panic.
		ctr, ctrErr := ot.Meter.Int64Counter("test_temp_counter", metric.WithDescription("temp"))
		if ctrErr != nil {
			t.Fatalf("failed to create temp counter: %v", ctrErr)
		}
		// Use a background context for recording; this should be a no-op and not panic.
		ctx := context.Background()
		// Ensure recording does not panic. Use testify's concise helper instead
		// of a manual defer/recover wrapper for clarity.
		require.NotPanics(t, func() { ctr.Add(ctx, 1) })
	})
}

func TestSetupOTelHistogramEachErrorBranch(t *testing.T) {
	histNames := []string{
		MetricReconcilePhaseDuration,
		MetricFetchDuration,
		MetricK8sRequestDuration,
		MetricGCRunDuration,
		MetricConfigParseDuration,
	}
	for _, target := range histNames {
		t.Run(target, func(t *testing.T) {
			withSeams(t, func() {
				// Fail only the targeted histogram; others succeed using the real meter.
				makeF64HistFn = func(m metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
					if name == target {
						return nil, errors.New("forced histogram error")
					}
					return m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
				}
				// Ensure counters and updown instruments succeed so we
				// deterministically reach the targeted histogram.
				successfulCtrAndUpDown()
				reg := prometheus.NewRegistry()
				ot, err := SetupOTel(reg)
				if err == nil || ot != nil {
					t.Fatalf("expected error when failing histogram %s, got ot=%v err=%v", target, ot, err)
				}
			})
		})
	}
}

func TestSetupOTelCounterEachErrorBranch(t *testing.T) {
	ctrNames := []string{
		MetricK8sRequestsTotal,
		MetricRetriesTotal,
		MetricLoopIterationsTotal,
	}
	for _, target := range ctrNames {
		t.Run(target, func(t *testing.T) {
			withSeams(t, func() {
				makeI64CtrFn = func(m metric.Meter, name, desc string) (metric.Int64Counter, error) {
					if name == target {
						return nil, errors.New("forced counter error")
					}
					return m.Int64Counter(name, metric.WithDescription(desc))
				}
				// Ensure preceding histograms and the updown instrument succeed so the
				// failing counter is the root cause exercised by this test.
				successfulHistAndUpDown()
				reg := prometheus.NewRegistry()
				ot, err := SetupOTel(reg)
				if err == nil || ot != nil {
					t.Fatalf("expected error when failing counter %s, got ot=%v err=%v", target, ot, err)
				}
			})
		})
	}
}
