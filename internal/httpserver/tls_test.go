package httpserver

import (
	"net"
	"strings"
	"testing"

	"pangolin-kube-controller/internal/config"
)

func TestComputeBindAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		addr string
		want string
	}{
		{
			name: "empty returns default",
			addr: "",
			want: ":9090",
		},
		{
			name: "port only prefixed with colon",
			addr: ":8080",
			want: net.IPv4zero.String() + ":8080",
		},
		{
			name: "full address unchanged",
			addr: "127.0.0.1:8080",
			want: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeBindAddr(tt.addr)
			if got != tt.want {
				t.Errorf("computeBindAddr(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestCheckPprofPlaintextAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "pprof disabled no error",
			cfg:     &config.Config{EnablePprof: false, MetricsPlaintextOK: false},
			wantErr: false,
		},
		{
			name:    "pprof enabled plaintext OK no error",
			cfg:     &config.Config{EnablePprof: true, MetricsPlaintextOK: true},
			wantErr: false,
		},
		{
			name:    "pprof enabled plaintext not OK returns error",
			cfg:     &config.Config{EnablePprof: true, MetricsPlaintextOK: false},
			wantErr: true,
			errMsg:  "pprof enabled but TLS is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkPprofPlaintextAllowed(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("checkPprofPlaintextAllowed() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("checkPprofPlaintextAllowed() error = %v, want containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("checkPprofPlaintextAllowed() unexpected error: %v", err)
				}
			}
		})
	}
}
