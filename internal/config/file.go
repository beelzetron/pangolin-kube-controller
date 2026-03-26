package config

import (
	"os"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"
	syaml "sigs.k8s.io/yaml"
)

type fileConfig struct {
	TraefikInstanceLabel       string `json:"traefikInstanceLabel" yaml:"traefikInstanceLabel"`
	TraefikInstanceLabelKey    string `json:"traefikInstanceLabelKey" yaml:"traefikInstanceLabelKey"`
	TraefikInstanceLabelValue  string `json:"traefikInstanceLabelValue" yaml:"traefikInstanceLabelValue"`
	IngressClass               string `json:"ingressClass" yaml:"ingressClass"`
	IngressClassLabelVerifyInt string `json:"ingressClassLabelVerifyInterval" yaml:"ingressClassLabelVerifyInterval"`
	IngressClassLabelStrict    *bool  `json:"ingressClassLabelStrict" yaml:"ingressClassLabelStrict"`
}

func mergeConfigFile(cfg *Config) {
	if cfg.ConfigFile == "" {
		return
	}
	b, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		logrus.Warnf("CONFIG_FILE=%s read error: %v", cfg.ConfigFile, err)
		return
	}
	var fc fileConfig
	if err := syaml.Unmarshal(b, &fc); err != nil {
		logrus.Warnf("CONFIG_FILE=%s parse error: %v", cfg.ConfigFile, err)
		return
	}
	mergeFileIngressClass(cfg, fc)
	mergeFileVerifyInterval(cfg, fc)
	mergeFileStrict(cfg, fc)
	mergeFileInstanceLabel(cfg, fc)
}

func mergeFileIngressClass(cfg *Config, fc fileConfig) {
	if cfg.IngressClassProvided {
		return
	}
	v := strings.TrimSpace(fc.IngressClass)
	if v == "" {
		return
	}
	cfg.IngressClass = v
	cfg.IngressClassProvided = true
}

func mergeFileVerifyInterval(cfg *Config, fc fileConfig) {
	if _, ok := os.LookupEnv("INGRESS_CLASS_LABEL_VERIFY_INTERVAL"); ok {
		return
	}
	raw := strings.TrimSpace(fc.IngressClassLabelVerifyInt)
	if raw == "" {
		return
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		logrus.Errorf("Invalid duration in config file ingressClassLabelVerifyInterval: %v", err)
		return
	}
	cfg.IngressClassLabelVerifyInterval = d
}

func mergeFileStrict(cfg *Config, fc fileConfig) {
	if _, ok := os.LookupEnv("INGRESS_CLASS_LABEL_STRICT"); ok {
		return
	}
	if fc.IngressClassLabelStrict == nil {
		return
	}
	cfg.IngressClassLabelStrict = *fc.IngressClassLabelStrict
}

func mergeFileInstanceLabel(cfg *Config, fc fileConfig) {
	if cfg.TraefikInstanceLabelKey != "" && cfg.TraefikInstanceLabelValue != "" {
		return
	}
	key := strings.TrimSpace(fc.TraefikInstanceLabelKey)
	val := strings.TrimSpace(fc.TraefikInstanceLabelValue)
	if key != "" && val != "" {
		cfg.TraefikInstanceLabelKey = key
		cfg.TraefikInstanceLabelValue = val
		return
	}
	single := strings.TrimSpace(fc.TraefikInstanceLabel)
	if single != "" {
		applyKVPair(cfg, single)
	}
}
