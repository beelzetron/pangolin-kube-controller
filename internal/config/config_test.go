package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tst "pangolin-kube-controller/internal/testutil"
)

const unexpectedErrFmt = "unexpected error: %v"

func TestEnvStringAndDefaults(t *testing.T) {
	if got := envString("__UNKNOWN__", "def"); got != "def" {
		t.Fatalf("envString default = %q", got)
	}
	withEnv(t, "FOO", "bar", func() {
		if got := envString("FOO", "def"); got != "bar" {
			t.Fatalf("envString override = %q", got)
		}
	})
}

func TestLoadFromEnvDefaults(t *testing.T) {
	withoutEnvMap(t, []string{"TARGET_NAMESPACE", "INGRESS_CLASS", "POLL_INTERVAL"}, func() {
		cfg := LoadFromEnv()
		if cfg.Namespace != tst.TestNamespace {
			t.Fatalf("default namespace mismatch")
		}
		if cfg.IngressClass != "traefik" {
			t.Fatalf("default ingress class mismatch")
		}
		if cfg.PollInterval <= 0 {
			t.Fatalf("poll interval not set")
		}
	})
}

func TestLoadFromEnvOverrides(t *testing.T) {
	withEnvMap(t, map[string]string{
		"TARGET_NAMESPACE":       "customns",
		"INGRESS_CLASS":          "customclass",
		"POLL_INTERVAL":          "2s",
		"ENABLE_LEADER_ELECTION": "true",
	}, func() {
		cfg := LoadFromEnv()
		if cfg.Namespace != "customns" {
			t.Fatalf("namespace override failed")
		}
		if cfg.IngressClass != "customclass" {
			t.Fatalf("ingress class override failed")
		}
		if cfg.PollInterval != 2*time.Second {
			t.Fatalf("poll interval override failed: %s", cfg.PollInterval)
		}
		if !cfg.LeaderEnabled {
			t.Fatalf("leader enabled override failed")
		}
	})
}

func TestEnvDurationValidInvalid(t *testing.T) {
	if got := envDuration("__DUR__", 5*time.Second); got != 5*time.Second {
		t.Fatalf("envDuration default = %v", got)
	}
	withEnv(t, "__DUR__", "250ms", func() {
		if got := envDuration("__DUR__", time.Second); got != 250*time.Millisecond {
			t.Fatalf("envDuration parsed = %v", got)
		}
	})
	withEnv(t, "__DUR__", "notaduration", func() {
		if got := envDuration("__DUR__", time.Second); got != time.Second {
			t.Fatalf("envDuration invalid fallback = %v", got)
		}
	})
}

func TestEnvBoolVariants(t *testing.T) {
	if got := envBool("__BOOL__", true); got != true {
		t.Fatalf("envBool default = %v", got)
	}
	for _, v := range []string{"true", "1", "yes", "TRUE"} {
		withEnv(t, "__BOOL__", v, func() {
			if !envBool("__BOOL__", false) {
				t.Fatalf("envBool true variant failed for %q", v)
			}
		})
	}
	for _, v := range []string{"false", "0", "no", "FALSE"} {
		withEnv(t, "__BOOL__", v, func() {
			if envBool("__BOOL__", true) {
				t.Fatalf("envBool false variant failed for %q", v)
			}
		})
	}
	withEnv(t, "__BOOL__", "maybe", func() {
		if !envBool("__BOOL__", true) {
			t.Fatalf("envBool invalid fallback")
		}
	})
}

func TestEnvIntAndFloat(t *testing.T) {
	if got := envInt("__INT__", 7); got != 7 {
		t.Fatalf("envInt default = %d", got)
	}
	withEnv(t, "__INT__", "42", func() {
		if got := envInt("__INT__", 7); got != 42 {
			t.Fatalf("envInt parsed = %d", got)
		}
	})
	withEnv(t, "__INT__", "nope", func() {
		if got := envInt("__INT__", 9); got != 9 {
			t.Fatalf("envInt invalid fallback = %d", got)
		}
	})

	if got := envFloat("__FLT__", 3.14); got != 3.14 {
		t.Fatalf("envFloat default = %v", got)
	}
	withEnv(t, "__FLT__", "2.5", func() {
		if got := envFloat("__FLT__", 0); got != 2.5 {
			t.Fatalf("envFloat parsed = %v", got)
		}
	})
	withEnv(t, "__FLT__", "bad", func() {
		if got := envFloat("__FLT__", 1.23); got != 1.23 {
			t.Fatalf("envFloat invalid fallback = %v", got)
		}
	})
}

func TestPopulateInstanceLabelFromEnvPairAndSingle(t *testing.T) {
	withEnvMap(t, map[string]string{"TRAEFIK_INSTANCE_LABEL": "app=demo"}, func() {
		cfg := newDefaults()
		populateInstanceLabelFromEnv(cfg)
		if cfg.TraefikInstanceLabelKey != "app" || cfg.TraefikInstanceLabelValue != "demo" {
			t.Fatalf("single env failed: %s=%s", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
		}
	})

	withEnvMap(t, map[string]string{"TRAEFIK_INSTANCE_LABEL_KEY": "k", "TRAEFIK_INSTANCE_LABEL_VALUE": "v"}, func() {
		cfg := newDefaults()
		populateInstanceLabelFromEnv(cfg)
		if cfg.TraefikInstanceLabelKey != "k" || cfg.TraefikInstanceLabelValue != "v" {
			t.Fatalf("pair precedence failed: %s=%s", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
		}
	})
}

func TestApplyKVPairAndMergeFileInstanceLabel(t *testing.T) {
	cfg := &Config{}
	applyKVPair(cfg, " key = value ")
	if cfg.TraefikInstanceLabelKey != "key" || cfg.TraefikInstanceLabelValue != "value" {
		t.Fatalf("applyKVPair failed: %q=%q", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
	}

	cfg = &Config{}
	applyKVPair(cfg, "noequals")
	if cfg.TraefikInstanceLabelKey != "" || cfg.TraefikInstanceLabelValue != "" {
		t.Fatalf("applyKVPair should ignore invalid pair")
	}

	cfg = &Config{}
	fc := fileConfig{TraefikInstanceLabelKey: "  mykey  ", TraefikInstanceLabelValue: "  myval  "}
	mergeFileInstanceLabel(cfg, fc)
	if cfg.TraefikInstanceLabelKey != "mykey" || cfg.TraefikInstanceLabelValue != "myval" {
		t.Fatalf("mergeFileInstanceLabel explicit failed: %q=%q", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
	}

	cfg = &Config{TraefikInstanceLabelKey: "exist", TraefikInstanceLabelValue: "existv"}
	fc = fileConfig{TraefikInstanceLabel: "fileonly=val"}
	mergeFileInstanceLabel(cfg, fc)
	if cfg.TraefikInstanceLabelKey != "exist" || cfg.TraefikInstanceLabelValue != "existv" {
		t.Fatalf("mergeFileInstanceLabel should not override existing")
	}

	cfg = &Config{}
	fc = fileConfig{TraefikInstanceLabel: "  a=b  "}
	mergeFileInstanceLabel(cfg, fc)
	if cfg.TraefikInstanceLabelKey != "a" || cfg.TraefikInstanceLabelValue != "b" {
		t.Fatalf("mergeFileInstanceLabel combined failed: %q=%q", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
	}
}

func TestMergeFileVerifyInterval(t *testing.T) {
	cfg := &Config{}
	fc := fileConfig{IngressClassLabelVerifyInt: " 1h "}
	mergeFileVerifyInterval(cfg, fc)
	if cfg.IngressClassLabelVerifyInterval != time.Hour {
		t.Fatalf("expected 1h, got %v", cfg.IngressClassLabelVerifyInterval)
	}

	cfg = &Config{IngressClassLabelVerifyInterval: 2 * time.Hour}
	fc = fileConfig{IngressClassLabelVerifyInt: "notaduration"}
	mergeFileVerifyInterval(cfg, fc)
	if cfg.IngressClassLabelVerifyInterval != 2*time.Hour {
		t.Fatalf("expected preserved value on parse error")
	}
}

func TestMergeConfigFileAppliesFields(t *testing.T) {
	p := writeTempConfig(t, ""+
		"ingressClass: ic-x\n"+
		"ingressClassLabelStrict: true\n"+
		"ingressClassLabelVerifyInterval: 1h\n"+
		"traefikInstanceLabelKey: app.kubernetes.io/instance\n"+
		"traefikInstanceLabelValue: demo\n",
	)
	cfg := newDefaults()
	cfg.ConfigFile = p
	mergeConfigFile(cfg)
	if !cfg.IngressClassProvided || cfg.IngressClass != "ic-x" {
		t.Fatalf("ingress class not applied: provided=%v val=%s", cfg.IngressClassProvided, cfg.IngressClass)
	}
	if !cfg.IngressClassLabelStrict {
		t.Fatalf("strict not applied")
	}
	if cfg.IngressClassLabelVerifyInterval != time.Hour {
		t.Fatalf("verify interval not applied: %s", cfg.IngressClassLabelVerifyInterval)
	}
	if cfg.TraefikInstanceLabelKey != "app.kubernetes.io/instance" || cfg.TraefikInstanceLabelValue != "demo" {
		t.Fatalf("instance label not applied: %s=%s", cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
	}
}

func TestMergeFileIngressClassEnvPrecedence(t *testing.T) {
	withEnv(t, "INGRESS_CLASS", "from-env", func() {
		cfg := newDefaults()
		markIngressClassProvided(cfg)
		p := writeTempConfig(t, "ingressClass: from-file\n")
		cfg.ConfigFile = p
		mergeConfigFile(cfg)
		if cfg.IngressClass != "from-env" {
			t.Fatalf("env precedence not respected: got %s", cfg.IngressClass)
		}
	})
}

func TestDefaultConfigEndpointIsHTTPS(t *testing.T) {
	t.Setenv("CONFIG_ENDPOINT", "")
	cfg := LoadFromEnv()
	if len(cfg.Endpoint) < 8 || cfg.Endpoint[:8] != "https://" {
		t.Fatalf("default CONFIG_ENDPOINT must use https://, got %q", cfg.Endpoint)
	}
}

func TestAllowInsecureHTTPEnvVarTrue(t *testing.T) {
	withEnv(t, "CONFIG_ALLOW_INSECURE_HTTP", "true", func() {
		cfg := LoadFromEnv()
		if !cfg.AllowInsecureHTTP {
			t.Fatalf("expected AllowInsecureHTTP=true when CONFIG_ALLOW_INSECURE_HTTP=true")
		}
	})
}

func TestAllowInsecureHTTPEnvVarFalse(t *testing.T) {
	withEnv(t, "CONFIG_ALLOW_INSECURE_HTTP", "false", func() {
		cfg := LoadFromEnv()
		if cfg.AllowInsecureHTTP {
			t.Fatalf("expected AllowInsecureHTTP=false when CONFIG_ALLOW_INSECURE_HTTP=false")
		}
	})
}

func TestAllowInsecureHTTPDefaultFalse(t *testing.T) {
	t.Setenv("CONFIG_ALLOW_INSECURE_HTTP", "")
	cfg := LoadFromEnv()
	if cfg.AllowInsecureHTTP {
		t.Fatalf("expected AllowInsecureHTTP=false by default")
	}
}

func TestAllowInsecureHTTPEnvVarVariants(t *testing.T) {
	trueVariants := []string{"1", "true", "TRUE", "True", "yes", "YES", "Yes"}
	for _, val := range trueVariants {
		val := val
		t.Run("true_"+val, func(t *testing.T) {
			withEnv(t, "CONFIG_ALLOW_INSECURE_HTTP", val, func() {
				cfg := LoadFromEnv()
				if !cfg.AllowInsecureHTTP {
					t.Fatalf("expected AllowInsecureHTTP=true for value %q", val)
				}
			})
		})
	}

	falseVariants := []string{"0", "false", "FALSE", "False", "no", "NO", "No"}
	for _, val := range falseVariants {
		val := val
		t.Run("false_"+val, func(t *testing.T) {
			withEnv(t, "CONFIG_ALLOW_INSECURE_HTTP", val, func() {
				cfg := LoadFromEnv()
				if cfg.AllowInsecureHTTP {
					t.Fatalf("expected AllowInsecureHTTP=false for value %q", val)
				}
			})
		})
	}
}

func TestAllowInsecureHTTPInvalidValueDefaultsFalse(t *testing.T) {
	withEnv(t, "CONFIG_ALLOW_INSECURE_HTTP", "maybe", func() {
		cfg := LoadFromEnv()
		if cfg.AllowInsecureHTTP {
			t.Fatalf("expected AllowInsecureHTTP=false for invalid value")
		}
	})
}

func TestShouldLogConfigPreview(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{name: "nil config", cfg: nil, want: false},
		{name: "LogConfigPreview true", cfg: &Config{LogConfigPreview: true}, want: true},
		{name: "both flags true", cfg: &Config{LogConfigPreview: true, LogTraefikConfig: true}, want: true},
		{name: "LogTraefikConfig true", cfg: &Config{LogTraefikConfig: true}, want: true},
		{name: "both false", cfg: &Config{LogConfigPreview: false, LogTraefikConfig: false}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got bool
			if tc.cfg == nil {
				var c *Config
				got = c.ShouldLogConfigPreview()
			} else {
				got = tc.cfg.ShouldLogConfigPreview()
			}
			if got != tc.want {
				t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
			}
		})
	}
}

func TestNormalizeFetchLogInterval(t *testing.T) {
	t.Setenv("FETCH_LOG_INTERVAL", "")
	cfg := LoadFromEnv()
	cfg.FetchLogInterval = -1 * time.Second
	cfg.normalize()
	if cfg.FetchLogInterval != 0 {
		t.Fatalf("negative interval not clamped to 0: %v", cfg.FetchLogInterval)
	}
	cfg.FetchLogInterval = 48 * time.Hour
	cfg.normalize()
	if cfg.FetchLogInterval > 24*time.Hour {
		t.Fatalf("interval not clamped to 24h: %v", cfg.FetchLogInterval)
	}
}

func TestNormalizeFetchLogIntervalEnvNegative(t *testing.T) {
	withEnv(t, "FETCH_LOG_INTERVAL", "-5s", func() {
		cfg := LoadFromEnv()
		if cfg.FetchLogInterval != 0 {
			t.Fatalf("env negative not clamped to 0, got %v", cfg.FetchLogInterval)
		}
	})
}

func TestNormalizeClampEnvLoad(t *testing.T) {
	withEnv(t, "FETCH_LOG_INTERVAL", "48h", func() {
		cfg := LoadFromEnv()
		if cfg.FetchLogInterval > 24*time.Hour {
			t.Fatalf("fetch log interval not clamped: %s", cfg.FetchLogInterval)
		}
	})
}

func withEnv(t *testing.T, k, v string, fn func()) {
	t.Helper()
	t.Setenv(k, v)
	fn()
}

func withEnvMap(t *testing.T, vals map[string]string, fn func()) {
	t.Helper()
	for k, v := range vals {
		t.Setenv(k, v)
	}
	fn()
}

func withoutEnvMap(t *testing.T, keys []string, fn func()) {
	t.Helper()
	old := make(map[string]string, len(keys))
	had := make(map[string]bool, len(keys))
	for _, k := range keys {
		o, h := os.LookupEnv(k)
		old[k] = o
		had[k] = h
		// Unset the environment variable so callers relying on os.LookupEnv see it absent
		_ = os.Unsetenv(k)
	}
	// Ensure restoration of original environment state even if fn panics
	defer func() {
		for _, k := range keys {
			if had[k] {
				_ = os.Setenv(k, old[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()
	fn()
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp cfg: %v", err)
	}
	return p
}

func TestValidateOnLoseBehavior(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"exit is valid", "exit", false},
		{"pause is valid", "pause", false},
		{"EXIT uppercase valid", "EXIT", false},
		{"PAUSE mixed case valid", "Pause", false},
		{"invalid value", "bogus", true},
		{"unknown value", "stop", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{OnLoseBehavior: tt.value}
			err := cfg.validateOnLoseBehavior()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for ON_LOSE=%q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for ON_LOSE=%q: %v", tt.value, err)
			}
		})
	}
}

func TestValidateLeaseTiming(t *testing.T) {
	t.Run("valid timing", func(t *testing.T) {
		cfg := &Config{
			LeaderEnabled: true,
			LeaseDuration: 30 * time.Second,
			RenewDeadline: 20 * time.Second,
			RetryPeriod:   5 * time.Second,
		}
		if err := cfg.validateLeaseTiming(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})

	t.Run("LeaseDuration not greater than RenewDeadline", func(t *testing.T) {
		cfg := &Config{
			LeaderEnabled: true,
			LeaseDuration: 20 * time.Second,
			RenewDeadline: 20 * time.Second,
			RetryPeriod:   5 * time.Second,
		}
		if err := cfg.validateLeaseTiming(); err == nil {
			t.Fatal("expected error when LeaseDuration <= RenewDeadline")
		}
	})

	t.Run("RenewDeadline not greater than RetryPeriod", func(t *testing.T) {
		cfg := &Config{
			LeaderEnabled: true,
			LeaseDuration: 30 * time.Second,
			RenewDeadline: 5 * time.Second,
			RetryPeriod:   5 * time.Second,
		}
		if err := cfg.validateLeaseTiming(); err == nil {
			t.Fatal("expected error when RenewDeadline <= RetryPeriod")
		}
	})

	t.Run("leader disabled skips validation", func(t *testing.T) {
		cfg := &Config{
			LeaderEnabled: false,
			LeaseDuration: 0,
			RenewDeadline: 0,
			RetryPeriod:   0,
		}
		if err := cfg.validateLeaseTiming(); err != nil {
			t.Fatalf("unexpected error when leader disabled: %v", err)
		}
	})
}

func TestValidateReconcileSettings(t *testing.T) {
	t.Run("parallel with low max", func(t *testing.T) {
		cfg := &Config{
			ReconcileParallel: true,
			ReconcileMax:      1,
		}
		if err := cfg.validateReconcileSettings(); err == nil {
			t.Fatal("expected error when ReconcileParallel with ReconcileMax < 2")
		}
	})

	t.Run("parallel with adequate max", func(t *testing.T) {
		cfg := &Config{
			ReconcileParallel: true,
			ReconcileMax:      4,
		}
		if err := cfg.validateReconcileSettings(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})

	t.Run("sequential no max required", func(t *testing.T) {
		cfg := &Config{
			ReconcileParallel: false,
			ReconcileMax:      1,
		}
		if err := cfg.validateReconcileSettings(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})
}

func TestValidateGCSettings(t *testing.T) {
	t.Run("valid queue size", func(t *testing.T) {
		cfg := &Config{GCGraceQueueSize: 256}
		if err := cfg.validateGCSettings(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})

	t.Run("zero queue size", func(t *testing.T) {
		cfg := &Config{GCGraceQueueSize: 0}
		if err := cfg.validateGCSettings(); err == nil {
			t.Fatal("expected error for zero queue size")
		}
	})

	t.Run("negative queue size", func(t *testing.T) {
		cfg := &Config{GCGraceQueueSize: -1}
		if err := cfg.validateGCSettings(); err == nil {
			t.Fatal("expected error for negative queue size")
		}
	})
}

func TestValidateBackoffSettings(t *testing.T) {
	t.Run("valid max backoff passes", func(t *testing.T) {
		cfg := &Config{MaxBackoff: 10 * time.Second}
		if err := cfg.validateBackoffSettings(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})

	t.Run("zero max backoff fails", func(t *testing.T) {
		cfg := &Config{MaxBackoff: 0}
		if err := cfg.validateBackoffSettings(); err == nil {
			t.Fatal("expected error for MAX_BACKOFF=0")
		}
	})

	t.Run("negative max backoff fails", func(t *testing.T) {
		cfg := &Config{MaxBackoff: -1 * time.Second}
		if err := cfg.validateBackoffSettings(); err == nil {
			t.Fatal("expected error for negative MAX_BACKOFF")
		}
	})
}

func TestValidateAll(t *testing.T) {
	t.Run("valid config passes all", func(t *testing.T) {
		cfg := &Config{
			OnLoseBehavior:    "pause",
			MaxBackoff:        2 * time.Minute,
			LeaderEnabled:     true,
			LeaseDuration:     30 * time.Second,
			RenewDeadline:     20 * time.Second,
			RetryPeriod:       5 * time.Second,
			ReconcileParallel: true,
			ReconcileMax:      4,
			GCGraceQueueSize:  256,
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf(unexpectedErrFmt, err)
		}
	})

	t.Run("invalid config fails", func(t *testing.T) {
		cfg := &Config{
			OnLoseBehavior:    "invalid",
			MaxBackoff:        2 * time.Minute,
			LeaderEnabled:     true,
			LeaseDuration:     30 * time.Second,
			RenewDeadline:     20 * time.Second,
			RetryPeriod:       5 * time.Second,
			ReconcileParallel: true,
			ReconcileMax:      4,
			GCGraceQueueSize:  256,
		}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error from Validate() when an underlying validator fails")
		}
	})
}
