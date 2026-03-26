package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthcheckOK(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	os.Args = []string{"healthcheck", ts.URL}
	code := -1
	oldExit := osExit
	osExit = func(c int) { code = c }
	defer func() { osExit = oldExit }()

	main()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestHealthcheck503(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	os.Args = []string{"healthcheck", ts.URL}
	code := -1
	oldExit := osExit
	osExit = func(c int) { code = c }
	defer func() { osExit = oldExit }()

	main()
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestHealthcheckError(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Invalid URL to force error
	os.Args = []string{"healthcheck", "http://*********:1/healthz"}
	code := -1
	oldExit := osExit
	osExit = func(c int) { code = c }
	defer func() { osExit = oldExit }()

	main()
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestHealthcheckEnvAddr(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Extract host:port from ts.URL
	addr := ts.URL[len("http://"):]
	t.Setenv("METRICS_ADDR", addr)
	os.Args = []string{"healthcheck"}

	code := -1
	oldExit := osExit
	osExit = func(c int) { code = c }
	defer func() { osExit = oldExit }()

	main()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
