// Package tracing provides distributed tracing support for the controller.
//
// This package is reserved for future use. OpenTelemetry tracing is currently
// integrated via the otel SDK in internal/observability/metrics_otel/.
// Distributed tracing via an explicit TracerProvider is not yet wired.
// Tracing helpers may be added here when a concrete tracing exporter
// (OTLP/Console/etc.) is formally adopted.
package tracing
