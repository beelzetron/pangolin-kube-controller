package httpserver

import (
	"fmt"
	"net"
	"strings"

	"github.com/sirupsen/logrus"

	"pangolin-kube-controller/internal/config"
)

func computeBindAddr(addr string) string {
	if addr == "" {
		return ":9090"
	}
	if strings.HasPrefix(addr, ":") {
		return net.IPv4zero.String() + addr
	}
	return addr
}

func checkPprofPlaintextAllowed(cfg *config.Config) error {
	if cfg.EnablePprof && !cfg.MetricsPlaintextOK {
		return fmt.Errorf("pprof enabled but TLS is not configured and METRICS_PLAINTEXT_OK=false; refusing to start metrics/pprof over plaintext")
	}
	return nil
}

func logServerStart(addr string, tls bool) {
	if tls {
		logrus.Infof("metrics server starting with TLS on %s", addr)
	} else {
		logrus.Infof("metrics server starting on %s", addr)
	}
}
