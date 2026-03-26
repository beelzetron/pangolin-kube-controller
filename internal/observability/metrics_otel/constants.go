package metrics_otel

const (
	MetricReconcilePhaseDuration = "pangolin_controller_reconcile_phase_duration_seconds"
	MetricFetchDuration          = "pangolin_controller_fetch_duration_seconds"
	MetricK8sRequestDuration     = "pangolin_controller_k8s_request_duration_seconds"
	MetricGCRunDuration          = "pangolin_controller_gc_run_duration_seconds"
	MetricConfigParseDuration    = "pangolin_controller_config_parse_duration_seconds"

	MetricK8sRequestsTotal    = "pangolin_controller_k8s_requests_total"
	MetricRetriesTotal        = "pangolin_controller_retries_total"
	MetricLoopIterationsTotal = "pangolin_controller_loop_iterations_total"
)
