package protocol

import (
	"testing"

	"pangolin-kube-controller/internal/config"
)

func TestGetTraefikEnvURL(t *testing.T) {
	cfg := &config.Config{}
	cfg.TraefikLBURL = "https://x.y:1234"
	if got := getTraefikEnvURL(cfg); got != "https://x.y:1234" {
		t.Fatalf("TraefikLBURL not preferred: %q", got)
	}
	cfg.TraefikLBURL = ""
	cfg.TraefikLBScheme = "http"
	cfg.TraefikLBIP = "1.2.3.4" //NOSONAR
	cfg.TraefikLBPort = ""
	if got := getTraefikEnvURL(cfg); got != "http://1.2.3.4" { //NOSONAR
		t.Fatalf("derived URL missing: %q", got)
	}
	cfg.TraefikLBPort = "8080"
	if got := getTraefikEnvURL(cfg); got != "http://1.2.3.4:8080" { //NOSONAR
		t.Fatalf("derived URL with port missing: %q", got)
	}
	cfg.TraefikLBIP = ""
	if got := getTraefikEnvURL(cfg); got != "" {
		t.Fatalf("expected empty when IP missing: %q", got)
	}
}
