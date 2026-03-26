package httpserver

import (
	"net/http"
	"net/http/pprof"
	"time"

	"pangolin-kube-controller/internal/config"
)

func newServeMux(cfg *config.Config, metricsHandler http.Handler, readiness func() bool) *http.ServeMux {
	mux := http.NewServeMux()

	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}

	if !cfg.DisableLivez {
		mux.HandleFunc("/livez", livezHandler)
		mux.HandleFunc("/health/live", livezHandler)
	}

	mux.HandleFunc("/healthz", readyHandler(readiness))
	mux.HandleFunc("/readyz", readyHandler(readiness))
	mux.HandleFunc("/health/ready", readyHandler(readiness))

	if cfg.EnablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return mux
}

func livezHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func readyHandler(readiness func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if readiness() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	}
}

func timeoutOrDefault(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}
