package metrics_prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test basic metric increments and handler exposure.
func TestCollectorMetrics(t *testing.T) {
	c := NewCollector()
	c.ReconcileDuration.Observe(0.42)
	c.ReconcileErrors.Inc()
	c.AppliedObjects.WithLabelValues("IngressRouteTCP", "create").Inc()
	c.DeletedObjects.WithLabelValues("IngressRouteTCP").Inc()
	c.Ready.Set(1)
	c.ConsecutiveErrors.Set(2)
	c.LastFetchSuccess.Set(1234567890)
	c.DesiredObjects.WithLabelValues("Service").Set(5)
	c.GCDeletedTotal.WithLabelValues("Service", "stale").Add(3)
	c.GCRunsTotal.WithLabelValues("success").Inc()

	// Ensure handler renders without panic and includes a sample metric.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	c.Handler().ServeHTTP(rr, req)
	body := rr.Body.String()
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	for _, needle := range []string{
		"pangolin_kube_controller_reconcile_seconds",
		"pangolin_kube_controller_reconcile_errors_total",
		"pangolin_kube_controller_ready",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("metrics output missing %s; sample: %s", needle, body[:min(300, len(body))])
		}
	}
}
