package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// seams for testability
var (
	osExit  = os.Exit
	httpGet = func(url string) (*http.Response, error) {
		client := &http.Client{Timeout: 2 * time.Second}
		return client.Get(url)
	}
)

const defaultHost = "localhost"

func main() {
	flag.Parse()
	arg := ""
	if flag.NArg() > 0 {
		arg = flag.Arg(0)
	}
	// If an explicit HTTP URL is passed, probe it; otherwise build one from METRICS_ADDR.
	url := strings.TrimSpace(arg)
	if url == "" || !strings.HasPrefix(url, "http") {
		addr := os.Getenv("METRICS_ADDR")
		if addr == "" {
			addr = ":9090"
		}
		if strings.HasPrefix(addr, ":") {
			addr = defaultHost + addr
		}
		url = fmt.Sprintf("http://%s/health/ready", addr)
	}
	resp, err := httpGet(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health HTTP GET failed: %v\n", err)
		osExit(1)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "unhealthy status: %d\n", resp.StatusCode)
		osExit(1)
		return
	}
	osExit(0)
}
