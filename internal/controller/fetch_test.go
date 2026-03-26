package controller

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"pangolin-kube-controller/internal/config"
)

const (
	testAuthHeader  = "Bearer abc"
	testURLExample  = "https://example.com"
	testURLAttacker = "https://attacker.com"
)

func TestBuildHTTPClientRedirectCopiesHeaders(t *testing.T) {
	require := require.New(t)
	c := newCtrlForTest()
	c.cfg.AuthHeader = testAuthHeader
	cl := c.buildHTTPClientFromConfig(c.cfg)
	req, _ := http.NewRequest("GET", testURLExample, http.NoBody)
	viaReq, _ := http.NewRequest("GET", testURLExample, http.NoBody)
	viaReq.Header.Set("If-None-Match", "\"etag\"")
	require.NoError(cl.CheckRedirect(req, []*http.Request{viaReq}), "unexpected redirect error")
	require.Equal(testAuthHeader, req.Header.Get("Authorization"), "Authorization not propagated")
	require.Equal("\"etag\"", req.Header.Get("If-None-Match"), "If-None-Match not propagated")
}

func TestRedirectDoesNotForwardAuthAcrossHosts(t *testing.T) {
	require := require.New(t)
	c := newCtrlForTest()
	c.cfg.AuthHeader = testAuthHeader
	cl := c.buildHTTPClientFromConfig(c.cfg)
	newReq, _ := http.NewRequest("GET", testURLAttacker, http.NoBody)
	viaReq, _ := http.NewRequest("GET", testURLExample, http.NoBody)
	require.NoError(cl.CheckRedirect(newReq, []*http.Request{viaReq}), "unexpected redirect error")
	require.Empty(newReq.Header.Get("Authorization"), "Authorization should not be forwarded across hosts")
}

func TestNewHTTPTransportValues(t *testing.T) {
	require := require.New(t)
	c := newCtrlForTest()
	c.cfg.HTTPMaxIdleConns = 123
	c.cfg.HTTPMaxIdleConnsPerHost = 7
	c.cfg.HTTPIdleConnTimeout = 42 * time.Second
	tr := c.newHTTPTransport(c.cfg, c.buildTLSConfigFromConfig(c.cfg))
	require.Equal(123, tr.MaxIdleConns)
	require.Equal(7, tr.MaxIdleConnsPerHost)
	require.Equal(42*time.Second, tr.IdleConnTimeout)
}

func TestApplySkipVerifyBooleanLogicVariants(t *testing.T) {
	testCases := []struct {
		name      string
		envValue  string
		shouldSet bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"True mixed", "True", true},
		{"1 numeric", "1", true},
		{"yes lowercase", "yes", true},
		{"YES uppercase", "YES", true},
		{"Yes mixed", "Yes", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"invalid", "maybe", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{TLSSkipVerify: true}
			tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
			applySkipVerifyWithAck(cfg, tlsCfg, tc.envValue)

			if tc.shouldSet {
				if !tlsCfg.InsecureSkipVerify {
					t.Fatalf("should set InsecureSkipVerify for %q", tc.envValue)
				}
			} else {
				if tlsCfg.InsecureSkipVerify {
					t.Fatalf("should not set InsecureSkipVerify for %q", tc.envValue)
				}
			}
		})
	}
}

func TestApplySkipVerifyConfigFalseNeverSets(t *testing.T) {
	require := require.New(t)

	cfg := &config.Config{TLSSkipVerify: false}
	t.Setenv("I_UNDERSTAND_CONFIG_TLS_SKIP_VERIFY_IS_INSECURE", "true")

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
	applySkipVerify(cfg, tlsCfg)

	require.False(tlsCfg.InsecureSkipVerify, "should not set when cfg.TLSSkipVerify is false")
}

func TestApplySkipVerifyAllConditionsRequired(t *testing.T) {
	cfg := &config.Config{TLSSkipVerify: true}

	validValues := []string{"true", "1", "yes", "TRUE", "YES"}
	for _, val := range validValues {
		t.Run(val, func(t *testing.T) {
			t.Setenv("I_UNDERSTAND_CONFIG_TLS_SKIP_VERIFY_IS_INSECURE", val)
			tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
			applySkipVerify(cfg, tlsCfg)
			require.True(t, tlsCfg.InsecureSkipVerify, "should set for value: %q", val)
		})
	}

	invalidValues := []string{"true1", "yes1", "maybe", "ok"}
	for _, val := range invalidValues {
		t.Run(val, func(t *testing.T) {
			t.Setenv("I_UNDERSTAND_CONFIG_TLS_SKIP_VERIFY_IS_INSECURE", val)
			tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
			applySkipVerify(cfg, tlsCfg)
			require.False(t, tlsCfg.InsecureSkipVerify, "should not set for invalid value: %q", val)
		})
	}
}

func TestReadWithLimitNoLimit(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)
	got, _, err := readWithLimit(r, 0)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestReadWithLimitWithinBounds(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)
	got, _, err := readWithLimit(r, int64(len(data)+10))
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestReadWithLimitExceeded(t *testing.T) {
	data := strings.Repeat("x", 100)
	r := strings.NewReader(data)
	_, _, err := readWithLimit(r, 10)
	require.Error(t, err, "expected error when body exceeds limit")
	require.Contains(t, err.Error(), "exceeds maximum size")
}

func TestReadWithLimitExactBoundary(t *testing.T) {
	data := []byte("abcde")
	r := bytes.NewReader(data)
	got, _, err := readWithLimit(r, int64(len(data)))
	require.NoError(t, err)
	require.Equal(t, data, got)
}
