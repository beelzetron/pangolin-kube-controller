package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"pangolin-kube-controller/internal/config"
)

// Server hosts the lightweight HTTP endpoints used by the controller, such as
// health, readiness and metrics. It wraps an `http.Server` and exposes
// lifecycle helpers used by the application to start, serve and shutdown the
// instrumented HTTP handlers.
type Server struct {
	cfg       *config.Config
	srv       *http.Server
	readiness func() bool
}

// NewServer constructs a Server configured from cfg and the provided metrics
// handler. The handler may be nil to disable the /metrics endpoint.
func NewServer(cfg *config.Config, metricsHandler http.Handler) *Server {
	s := &Server{
		cfg:       cfg,
		readiness: func() bool { return false },
		srv: &http.Server{
			Addr:              cfg.MetricsAddr,
			ReadHeaderTimeout: timeoutOrDefault(cfg.MetricsReadHeaderTimeout, 3*time.Second),
		},
	}

	mux := newServeMux(cfg, metricsHandler, func() bool { return s.readiness() })
	s.srv.Handler = mux

	return s
}

func (s *Server) SetReadinessFunc(fn func() bool) {
	if fn == nil {
		s.readiness = func() bool { return false }
		return
	}
	s.readiness = fn
}

func (s *Server) Start() error {
	if s.cfg.MetricsTLSCertFile != "" && s.cfg.MetricsTLSKeyFile != "" {
		s.srv.Addr = s.cfg.MetricsAddr
		logServerStart(s.cfg.MetricsAddr, true)
		return s.srv.ListenAndServeTLS(s.cfg.MetricsTLSCertFile, s.cfg.MetricsTLSKeyFile)
	}

	if err := checkPprofPlaintextAllowed(s.cfg); err != nil {
		return err
	}

	bindAddr := computeBindAddr(s.cfg.MetricsAddr)
	logServerStart(bindAddr, false)
	s.srv.Addr = bindAddr
	return s.srv.ListenAndServe()
}

func (s *Server) Serve(ln net.Listener) error {
	if s.cfg.MetricsTLSCertFile != "" && s.cfg.MetricsTLSKeyFile != "" {
		s.srv.Addr = s.cfg.MetricsAddr
		logServerStart(s.cfg.MetricsAddr, true)
		return s.srv.ServeTLS(ln, s.cfg.MetricsTLSCertFile, s.cfg.MetricsTLSKeyFile)
	}

	if err := checkPprofPlaintextAllowed(s.cfg); err != nil {
		return err
	}
	addr := ln.Addr().String()
	logServerStart(addr, false)
	s.srv.Addr = addr
	return s.srv.Serve(ln)
}

// Shutdown gracefully stops the HTTP server within the provided context
// timeout, returning any shutdown error wrapped with context.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return fmt.Errorf("shutdown metrics server: %w", err)
	}
	return nil
}
