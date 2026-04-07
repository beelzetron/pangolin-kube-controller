package controller

import (
	"context"
	"time"
)

func (c *Controller) Ready() bool {
	if c.paused.Load() {
		c.setReady(0)
		return false
	}
	ts := c.lastSuccessfulReconcile.Load()
	if ts == 0 {
		c.setReady(0)
		return false
	}
	last := time.Unix(0, ts)
	pi := time.Second
	if c.cfg != nil && c.cfg.PollInterval > 0 {
		pi = c.cfg.PollInterval
	}
	if time.Since(last) > 5*pi {
		c.setReady(0)
		return false
	}
	c.setReady(1)
	return true
}

func (c *Controller) setReady(value float64) {
	if c.collector != nil {
		c.collector.Ready.Set(value)
	}
}

func (*Controller) Standalone(ctx context.Context) {
	<-ctx.Done()
}
