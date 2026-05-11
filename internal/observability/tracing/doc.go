// Package tracing provides distributed tracing support for the controller.
//
// This package integrates with OpenTelemetry for distributed traces. It
// instruments key controller components such as HTTP handlers, background
// reconcile jobs, Kubernetes API calls, and other long-running operations.
//
// Usage (high level): install an exporter (OTLP/Console/etc.), create and
// register an OpenTelemetry TracerProvider, then obtain tracers with
// `otel.Tracer("pangolin-controller")` and start spans around operations.
// See package-level constructors in this folder for concrete setup helpers.
package tracing
