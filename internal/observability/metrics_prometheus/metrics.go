package metrics_prometheus

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	logrus "github.com/sirupsen/logrus"

	otelmetrics "pangolin-kube-controller/internal/observability/metrics_otel"
)

// Collector owns the Prometheus registry and all controller metrics. When
// OpenTelemetry is configured it also exposes matching instruments via the
// Prometheus exporter.
type Collector struct {
	Registry          *prometheus.Registry
	ReconcileDuration prometheus.Histogram
	ReconcileErrors   prometheus.Counter
	AppliedObjects    *prometheus.CounterVec
	DeletedObjects    *prometheus.CounterVec
	Ready             prometheus.Gauge
	ConsecutiveErrors prometheus.Gauge
	LastFetchSuccess  prometheus.Gauge
	DesiredObjects    *prometheus.GaugeVec
	GCDeletedTotal    *prometheus.CounterVec
	GCRunsTotal       *prometheus.CounterVec

	// Instance-label related metrics
	InstanceLabelDetectSuccess  prometheus.Counter
	InstanceLabelDetectFailures prometheus.Counter
	InstanceLabelLastCheck      prometheus.Gauge

	// OpenTelemetry instruments (exported via Prometheus exporter on the same registry)
	OTel *otelmetrics.OTel
}

// ShutdownOTel flushes and shuts down the OpenTelemetry MeterProvider if available.
func (c *Collector) ShutdownOTel(ctx context.Context) error {
	if c.OTel == nil {
		return nil
	}
	return c.OTel.Shutdown(ctx)
}

// NewCollector creates a Collector, registers metrics and sets up the optional
// OpenTelemetry Prometheus exporter.
func NewCollector() *Collector {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	c := &Collector{
		Registry: reg,
		ReconcileDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "pangolin_kube_controller_reconcile_seconds",
			Help:    "Duration of a full successful reconcile loop",
			Buckets: prometheus.DefBuckets,
		}),
		ReconcileErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_reconcile_errors_total",
			Help: "Total errors during reconcile steps",
		}),
		AppliedObjects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_objects_applied_total",
			Help: "Applied (create/patch) objects by kind",
		}, []string{"kind", "action"}),
		DeletedObjects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_objects_deleted_total",
			Help: "Deleted objects by kind",
		}, []string{"kind"}),
		Ready: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pangolin_kube_controller_ready",
			Help: "Readiness state of the controller (1=ready,0=not ready)",
		}),
		ConsecutiveErrors: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pangolin_kube_controller_consecutive_errors",
			Help: "Number of consecutive reconcile errors",
		}),
		LastFetchSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pangolin_kube_controller_last_fetch_success_timestamp_seconds",
			Help: "Unix timestamp of last successful fetch",
		}),
		DesiredObjects: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pangolin_kube_controller_desired_objects_count",
			Help: "Desired objects count by kind",
		}, []string{"kind"}),
		GCDeletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_gc_deleted_total",
			Help: "GC deleted objects by kind and reason",
		}, []string{"kind", "reason"}),
		GCRunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_gc_runs_total",
			Help: "GC runs by result",
		}, []string{"result"}),

		// Instance-label metrics
		InstanceLabelDetectSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_instance_label_detect_success_total",
			Help: "Successful instance label resolutions",
		}),
		InstanceLabelDetectFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pangolin_kube_controller_instance_label_detect_failure_total",
			Help: "Failed instance label resolutions/verification",
		}),
		InstanceLabelLastCheck: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pangolin_kube_controller_instance_label_last_check_timestamp_seconds",
			Help: "Unix timestamp of last instance label verification",
		}),
	}

	reg.MustRegister(
		c.ReconcileDuration,
		c.ReconcileErrors,
		c.AppliedObjects,
		c.DeletedObjects,
		c.Ready,
		c.ConsecutiveErrors,
		c.LastFetchSuccess,
		c.DesiredObjects,
		c.GCDeletedTotal,
		c.GCRunsTotal,
		c.InstanceLabelDetectSuccess,
		c.InstanceLabelDetectFailures,
		c.InstanceLabelLastCheck,
	)

	// Initialize OpenTelemetry metric exporter and instruments on the same registry.
	if ot, err := otelmetrics.SetupOTel(reg); err != nil {
		logrus.Warnf("OpenTelemetry metrics disabled (setup failed): %v", err)
	} else {
		c.OTel = ot
	}

	return c
}

// Handler returns an http.Handler that serves the Prometheus registry.
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.Registry, promhttp.HandlerOpts{})
}
