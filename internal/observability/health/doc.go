// Package health provides health check endpoints for the controller.
//
// It exposes standard liveness and readiness endpoints (e.g., `/livez`,
// `/readyz`) used by Kubernetes to determine pod health and readiness for
// serving traffic. Handlers are lightweight and intended to be wired into the
// main HTTP server so orchestration systems can probe lifecycle state.
package health
