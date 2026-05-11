package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"
)

func markIngressClassProvided(cfg *Config) {
	if _, ok := os.LookupEnv("INGRESS_CLASS"); ok {
		cfg.IngressClassProvided = true
	}
}

func populateInstanceLabelFromEnv(cfg *Config) {
	k, okK := os.LookupEnv("TRAEFIK_INSTANCE_LABEL_KEY")
	v, okV := os.LookupEnv("TRAEFIK_INSTANCE_LABEL_VALUE")
	if okK && !okV {
		logrus.Warnf("TRAEFIK_INSTANCE_LABEL_KEY is set to %q but TRAEFIK_INSTANCE_LABEL_VALUE is missing; ignoring instance label pair", k)
		return
	}
	if !okK && okV {
		logrus.Warnf("TRAEFIK_INSTANCE_LABEL_VALUE is set to %q but TRAEFIK_INSTANCE_LABEL_KEY is missing; ignoring instance label pair", v)
		return
	}
	if okK && okV {
		cfg.TraefikInstanceLabelKey = strings.TrimSpace(k)
		cfg.TraefikInstanceLabelValue = strings.TrimSpace(v)
		return
	}
	if kv, ok := os.LookupEnv("TRAEFIK_INSTANCE_LABEL"); ok {
		applyKVPair(cfg, kv)
	}
}

func applyKVPair(cfg *Config, kv string) {
	parts := strings.SplitN(kv, "=", 2)
	if len(parts) == 2 {
		cfg.TraefikInstanceLabelKey = strings.TrimSpace(parts[0])
		cfg.TraefikInstanceLabelValue = strings.TrimSpace(parts[1])
	}
}

func envString(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			logrus.Errorf("Invalid duration for %s: %s, error: %v", k, v, err)
			return def
		}
		return d
	}
	return def
}

func envBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes") {
			return true
		}
		if strings.EqualFold(v, "false") || v == "0" || strings.EqualFold(v, "no") {
			return false
		}
		logrus.Errorf("Invalid bool for %s: %s", k, v)
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		iv, err := strconv.Atoi(v)
		if err != nil {
			logrus.Errorf("Invalid int for %s: %s", k, v)
			return def
		}
		return iv
	}
	return def
}

func envInt64(k string, def int64) int64 {
	if v := os.Getenv(k); v != "" {
		iv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			logrus.Errorf("Invalid int64 for %s: %s", k, v)
			return def
		}
		return iv
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		fv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			logrus.Errorf("Invalid float for %s: %s", k, v)
			return def
		}
		return fv
	}
	return def
}

// parseCertificateSecretsEnv parses a comma-separated list of Secret references
// of the form "namespace/secretName" or "secretName" (no namespace).
func parseCertificateSecretsEnv(raw string) []CertificateSecretRef {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var refs []CertificateSecretRef
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		slashCount := strings.Count(p, "/")
		if slashCount == 0 {
			refs = append(refs, CertificateSecretRef{SecretName: p})
			continue
		}
		if slashCount != 1 {
			logrus.Warnf("skipping token %q: invalid format (must be 'name' or 'namespace/name'), found %d slashes", p, slashCount)
			continue
		}
		ns, name, ok := strings.Cut(p, "/")
		if !ok {
			logrus.Warnf("skipping token %q: failed to split namespace and name", p)
			continue
		}
		ns = strings.TrimSpace(ns)
		name = strings.TrimSpace(name)
		if ns == "" {
			logrus.Warnf("skipping token %q: empty namespace", p)
			continue
		}
		if name == "" {
			logrus.Warnf("skipping token %q: empty secret name", p)
			continue
		}
		refs = append(refs, CertificateSecretRef{SecretName: name, Namespace: ns})
	}
	return refs
}
