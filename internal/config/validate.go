package config

import (
	"fmt"
	"strings"
)

func (c *Config) Validate() error {
	if err := c.validateOnLoseBehavior(); err != nil {
		return err
	}
	if err := c.validateBackoffSettings(); err != nil {
		return err
	}
	if err := c.validateLeaseTiming(); err != nil {
		return err
	}
	if err := c.validateReconcileSettings(); err != nil {
		return err
	}
	if err := c.validateGCSettings(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateBackoffSettings() error {
	if c.MaxBackoff <= 0 {
		return fmt.Errorf("MAX_BACKOFF must be positive (got %s)", c.MaxBackoff)
	}
	return nil
}

func (c *Config) validateOnLoseBehavior() error {
	switch strings.ToLower(c.OnLoseBehavior) {
	case "", "exit", "pause":
		return nil
	}
	return fmt.Errorf("ON_LOSE=%q is invalid: must be 'exit' or 'pause' (default: 'exit')", c.OnLoseBehavior)
}

func (c *Config) validateLeaseTiming() error {
	if !c.LeaderEnabled {
		return nil
	}
	if c.LeaseDuration <= 0 {
		return fmt.Errorf("LEASE_DURATION must be positive (got %s)", c.LeaseDuration)
	}
	if c.RenewDeadline <= 0 {
		return fmt.Errorf("RENEW_DEADLINE must be positive (got %s)", c.RenewDeadline)
	}
	if c.RetryPeriod <= 0 {
		return fmt.Errorf("RETRY_PERIOD must be positive (got %s)", c.RetryPeriod)
	}
	if c.LeaseDuration <= c.RenewDeadline {
		return fmt.Errorf("LEASE_DURATION (%s) must be greater than RENEW_DEADLINE (%s)", c.LeaseDuration, c.RenewDeadline)
	}
	if c.RenewDeadline <= c.RetryPeriod {
		return fmt.Errorf("RENEW_DEADLINE (%s) must be greater than RETRY_PERIOD (%s)", c.RenewDeadline, c.RetryPeriod)
	}
	return nil
}

func (c *Config) validateReconcileSettings() error {
	if c.ReconcileParallel && c.ReconcileMax < 2 {
		return fmt.Errorf("RECONCILE_MAX (%d) must be at least 2 when RECONCILE_PARALLEL is enabled", c.ReconcileMax)
	}
	return nil
}

func (c *Config) validateGCSettings() error {
	if c.GCGraceQueueSize <= 0 {
		return fmt.Errorf("GC_GRACE_QUEUE_SIZE must be positive (got %d)", c.GCGraceQueueSize)
	}
	return nil
}
