package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func mustUnsetenv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv %s: %v", key, err)
	}
}

func assertUnchangedInstanceLabel(t *testing.T, cfg *Config, orig *Config, kv string) {
	t.Helper()
	if cfg.TraefikInstanceLabelKey != orig.TraefikInstanceLabelKey || cfg.TraefikInstanceLabelValue != orig.TraefikInstanceLabelValue {
		t.Errorf("applyKVPair(%q) modified config unexpectedly", kv)
	}
}

func assertAppliedKVPair(t *testing.T, cfg *Config, kv string) {
	t.Helper()
	parts := strings.SplitN(kv, "=", 2)
	if len(parts) != 2 {
		return
	}
	wantKey := strings.TrimSpace(parts[0])
	wantVal := strings.TrimSpace(parts[1])
	if cfg.TraefikInstanceLabelKey != wantKey {
		t.Errorf("TraefikInstanceLabelKey = %q, want %q", cfg.TraefikInstanceLabelKey, wantKey)
	}
	if cfg.TraefikInstanceLabelValue != wantVal {
		t.Errorf("TraefikInstanceLabelValue = %q, want %q", cfg.TraefikInstanceLabelValue, wantVal)
	}
}

func TestEnvString(t *testing.T) {
	key := "TEST_STRING_" + t.Name()

	t.Setenv(key, "fromenv")
	got := envString(key, "default")
	if got != "fromenv" {
		t.Errorf("envString() = %q, want fromenv", got)
	}

	mustUnsetenv(t, key)
	got = envString(key, "default")
	if got != "default" {
		t.Errorf("envString() = %q, want default", got)
	}
}

func TestEnvDuration(t *testing.T) {
	key := "TEST_DURATION_" + t.Name()

	t.Setenv(key, "5s")
	got := envDuration(key, time.Second)
	if got != 5*time.Second {
		t.Errorf("envDuration() = %v, want 5s", got)
	}

	mustUnsetenv(t, key)
	got = envDuration(key, time.Second)
	if got != time.Second {
		t.Errorf("envDuration() = %v, want default 1s", got)
	}

	t.Setenv(key, "invalid")
	got = envDuration(key, time.Second)
	if got != time.Second {
		t.Errorf("envDuration() = %v, want default 1s for invalid", got)
	}
}

func TestEnvBool(t *testing.T) {
	key := "TEST_BOOL_" + t.Name()
	for _, tc := range []struct {
		val  string
		def  bool
		want bool
	}{
		{"true", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"false", true, false},
		{"0", true, false},
		{"no", true, false},
		{"", true, true},
		{"invalid", true, true},
	} {
		t.Setenv(key, tc.val)
		got := envBool(key, tc.def)
		if got != tc.want {
			t.Errorf("envBool(%q, %v) = %v, want %v", tc.val, tc.def, got, tc.want)
		}
	}
}

func TestEnvInt(t *testing.T) {
	key := "TEST_INT_" + t.Name()

	t.Setenv(key, "42")
	got := envInt(key, 10)
	if got != 42 {
		t.Errorf("envInt() = %d, want 42", got)
	}

	mustUnsetenv(t, key)
	got = envInt(key, 10)
	if got != 10 {
		t.Errorf("envInt() = %d, want default 10", got)
	}

	t.Setenv(key, "invalid")
	got = envInt(key, 10)
	if got != 10 {
		t.Errorf("envInt() = %d, want default 10 for invalid", got)
	}
}

func TestEnvInt64(t *testing.T) {
	key := "TEST_INT64_" + t.Name()

	t.Setenv(key, "42")
	got := envInt64(key, 10)
	if got != 42 {
		t.Errorf("envInt64() = %d, want 42", got)
	}

	mustUnsetenv(t, key)
	got = envInt64(key, 10)
	if got != 10 {
		t.Errorf("envInt64() = %d, want default 10", got)
	}

	t.Setenv(key, "invalid")
	got = envInt64(key, 10)
	if got != 10 {
		t.Errorf("envInt64() = %d, want default 10 for invalid", got)
	}
}

func TestEnvFloat(t *testing.T) {
	key := "TEST_FLOAT_" + t.Name()

	t.Setenv(key, "42.5")
	got := envFloat(key, 10.0)
	if got != 42.5 {
		t.Errorf("envFloat() = %f, want 42.5", got)
	}

	mustUnsetenv(t, key)
	got = envFloat(key, 10.0)
	if got != 10.0 {
		t.Errorf("envFloat() = %f, want default 10.0", got)
	}

	t.Setenv(key, "invalid")
	got = envFloat(key, 10.0)
	if got != 10.0 {
		t.Errorf("envFloat() = %f, want default 10.0 for invalid", got)
	}
}

func TestApplyKVPair(t *testing.T) {
	tests := []struct {
		name string
		kv   string
	}{
		{"normal key=value", "key=value"},
		{"key=value with spaces", "  key  =  value  "},
		{"key only no equals", "keyonly"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			orig := *cfg
			applyKVPair(cfg, tt.kv)
			if tt.kv == "" || !strings.Contains(tt.kv, "=") {
				assertUnchangedInstanceLabel(t, cfg, &orig, tt.kv)
				return
			}
			assertAppliedKVPair(t, cfg, tt.kv)
		})
	}
}

func TestPopulateInstanceLabelFromEnv(t *testing.T) {
	t.Setenv("TRAEFIK_INSTANCE_LABEL_KEY", "app")
	t.Setenv("TRAEFIK_INSTANCE_LABEL_VALUE", "myapp")

	cfg := &Config{}
	populateInstanceLabelFromEnv(cfg)
	if cfg.TraefikInstanceLabelKey != "app" {
		t.Errorf("TraefikInstanceLabelKey = %q, want app", cfg.TraefikInstanceLabelKey)
	}
	if cfg.TraefikInstanceLabelValue != "myapp" {
		t.Errorf("TraefikInstanceLabelValue = %q, want myapp", cfg.TraefikInstanceLabelValue)
	}
}

func TestMarkIngressClassProvided(t *testing.T) {
	t.Setenv("INGRESS_CLASS", "traefik")

	cfg := &Config{}
	markIngressClassProvided(cfg)
	if !cfg.IngressClassProvided {
		t.Error("IngressClassProvided should be true after os.LookupEnv")
	}
}

func TestMarkIngressClassProvidedUnset(t *testing.T) {
	mustUnsetenv(t, "INGRESS_CLASS")
	cfg := &Config{}
	markIngressClassProvided(cfg)
	if cfg.IngressClassProvided {
		t.Error("IngressClassProvided should be false when INGRESS_CLASS is not set")
	}
}
