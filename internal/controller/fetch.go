package controller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"

	"pangolin-kube-controller/internal/config"
	otelmetrics "pangolin-kube-controller/internal/observability/metrics_otel"

	"go.opentelemetry.io/otel/metric"
)

func (c *Controller) buildHTTPClientFromConfig(cfg *config.Config) *http.Client {
	tlsCfg := c.buildTLSConfigFromConfig(cfg)
	tr := c.newHTTPTransport(cfg, tlsCfg)
	client := &http.Client{Timeout: cfg.FetchTimeout, Transport: tr}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			if v := via[0].Header.Get(headerIfNoneMatch); v != "" {
				req.Header.Set(headerIfNoneMatch, v)
			}
			orig := via[0]
			sameHost := strings.EqualFold(req.URL.Host, orig.URL.Host)
			sameScheme := strings.EqualFold(req.URL.Scheme, orig.URL.Scheme)
			if cfg.AuthHeader != "" && sameHost && sameScheme {
				req.Header.Set(headerAuthorization, cfg.AuthHeader)
			} else {
				req.Header.Del(headerAuthorization)
			}
		}
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return nil
	}
	return client
}

func (*Controller) buildTLSConfigFromConfig(cfg *config.Config) *tls.Config {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	applySkipVerify(cfg, tlsCfg)
	loadCustomCA(cfg, tlsCfg)
	loadClientCert(cfg, tlsCfg)
	return tlsCfg
}

func applySkipVerify(cfg *config.Config, tlsCfg *tls.Config) {
	if !cfg.TLSSkipVerify {
		return
	}
	ack := os.Getenv("I_UNDERSTAND_CONFIG_TLS_SKIP_VERIFY_IS_INSECURE")
	applySkipVerifyWithAck(cfg, tlsCfg, ack)
}

func applySkipVerifyWithAck(cfg *config.Config, tlsCfg *tls.Config, ack string) {
	if !cfg.TLSSkipVerify {
		return
	}
	if !strings.EqualFold(ack, "true") && ack != "1" && !strings.EqualFold(ack, "yes") {
		logrus.Warn("CONFIG_TLS_SKIP_VERIFY=true ignored: set I_UNDERSTAND_CONFIG_TLS_SKIP_VERIFY_IS_INSECURE=true to proceed (INSECURE)")
		return
	}
	logrus.Warn("CONFIG_TLS_SKIP_VERIFY enabled with explicit acknowledgement: certificate verification disabled (INSECURE)")
	tlsCfg.InsecureSkipVerify = true
}

func loadCustomCA(cfg *config.Config, tlsCfg *tls.Config) {
	if cfg.CAFile == "" {
		return
	}
	pemData, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		logrus.Warnf("Failed to read CONFIG_CA_FILE=%s: %v", cfg.CAFile, err)
		return
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		logrus.Warnf("SystemCertPool() failed; falling back to NewCertPool(): %v", err)
	}
	if pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemData) {
		logrus.Warn("No certs appended from CONFIG_CA_FILE")
	}
	tlsCfg.RootCAs = pool
}

func loadClientCert(cfg *config.Config, tlsCfg *tls.Config) {
	if cfg.ClientCertFile == "" || cfg.ClientKeyFile == "" {
		return
	}
	cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
	if err != nil {
		logrus.Errorf("Failed to load client cert/key: %v", err)
		return
	}
	tlsCfg.Certificates = []tls.Certificate{cert}
}

func (*Controller) newHTTPTransport(cfg *config.Config, tlsCfg *tls.Config) *http.Transport {
	return &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConns:        cfg.HTTPMaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.HTTPIdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
	}
}

func (c *Controller) fetchConfigOnce(ctx context.Context, ifNoneMatch string) (string, []byte, int, error) {
	urlStr := c.cfg.Endpoint
	if strings.HasPrefix(strings.ToLower(urlStr), "http://") {
		allow := c.cfg.AllowInsecureHTTP || envBool("CONFIG_ALLOW_INSECURE_HTTP")
		if !allow {
			return "", nil, 0, fmt.Errorf("CONFIG_ENDPOINT %q uses plaintext HTTP which is disallowed; set CONFIG_ALLOW_INSECURE_HTTP=true to override (INSECURE)", urlStr)
		}
		logrus.Warn("CONFIG_ALLOW_INSECURE_HTTP=true: using plaintext HTTP for CONFIG_ENDPOINT; traffic is unencrypted (INSECURE)")
	}
	return c.fetchConditional(ctx, urlStr, ifNoneMatch)
}

func envBool(key string) bool {
	val := strings.ToLower(os.Getenv(key))
	return val == "true" || val == "1" || val == "yes"
}

func (c *Controller) fetchConditional(ctx context.Context, url string, ifNoneMatch string) (string, []byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return "", nil, 0, err
	}
	if c.cfg.AuthHeader != "" {
		req.Header.Set(headerAuthorization, c.cfg.AuthHeader)
	}
	if ifNoneMatch != "" {
		req.Header.Set(headerIfNoneMatch, ifNoneMatch)
	}
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.collector != nil && c.collector.OTel != nil {
			c.collector.OTel.FetchDuration.Record(ctx, time.Since(start).Seconds(),
				metric.WithAttributes(
					otelmetrics.AttrStatusClass.String(otelmetrics.StatusClass(0)),
				),
			)
		}
		return "", nil, 0, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logrus.Warnf("error closing response body: %v", cerr)
		}
	}()

	status := resp.StatusCode
	b, _, err := readWithLimit(resp.Body, c.cfg.MaxResponseBodyBytes)
	if c.collector != nil && c.collector.OTel != nil {
		c.collector.OTel.FetchDuration.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(
				otelmetrics.AttrStatusClass.String(otelmetrics.StatusClass(status)),
			),
		)
	}
	if err != nil {
		return "", nil, status, fmt.Errorf("read response body: %w", err)
	}
	etag := resp.Header.Get(headerETag)
	return etag, b, status, nil
}

func readWithLimit(r io.Reader, limit int64) ([]byte, int64, error) {
	if limit <= 0 {
		b, err := io.ReadAll(r)
		return b, 0, err
	}
	buf := make([]byte, 8192)
	var total int64
	var result []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
			total += int64(n)
			if total > limit {
				return result, total, fmt.Errorf("response body exceeds maximum size of %d bytes", limit)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return result, total, err
		}
	}
	return result, total, nil
}
