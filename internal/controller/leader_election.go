package controller

import (
	"context"

	logrus "github.com/sirupsen/logrus"
)

func (c *Controller) OnStartedLeading(ctx context.Context, id string) {
	logrus.Infof("Became leader: %s", id)
	c.paused.Store(false)
	if c.collector != nil {
		c.collector.LeaderState.Set(1)
	}
	runCtx := c.runCtx
	if runCtx == nil {
		runCtx = ctx
	}
	c.runLoop(runCtx)
	if runCtx.Err() != nil {
		logrus.Infof("Leadership ended for %s due to context cancellation: %v", id, runCtx.Err())
	}
}

func (c *Controller) OnStoppedLeading(cancel context.CancelFunc) {
	if c.collector != nil {
		c.collector.LeaderState.Set(0)
	}
	c.handleLeadershipLoss(cancel)
}

func onNewLeader(id, current string) {
	if current != id {
		logrus.Infof("New leader observed: %s", current)
	}
}
