package config

import (
	"time"

	logrus "github.com/sirupsen/logrus"
)

func (c *Config) ShouldLogConfigPreview() bool {
	return c != nil && (c.LogConfigPreview || c.LogTraefikConfig)
}

func (c *Config) normalize() {
	if c.FetchLogInterval < 0 {
		logrus.Warnf("FETCH_LOG_INTERVAL=%s is negative; disabling periodic info logs", c.FetchLogInterval)
		c.FetchLogInterval = 0
	}
	const maxFetchLogInterval = 24 * time.Hour
	if c.FetchLogInterval > maxFetchLogInterval {
		logrus.Warnf("FETCH_LOG_INTERVAL=%s exceeds 24h; clamping to %s", c.FetchLogInterval, maxFetchLogInterval)
		c.FetchLogInterval = maxFetchLogInterval
	}
}
