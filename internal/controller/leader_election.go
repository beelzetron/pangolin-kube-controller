package controller

import (
	"context"

	logrus "github.com/sirupsen/logrus"
)

func (c *Controller) OnStartedLeading(ctx context.Context, id string) {
	logrus.Infof("Became leader: %s", id)
	c.runLoop(ctx)
	if ctx.Err() != nil {
		logrus.Infof("Leadership ended for %s due to context cancellation: %v", id, ctx.Err())
	}
}

func (c *Controller) OnStoppedLeading(cancel context.CancelFunc) {
	c.handleLeadershipLoss(cancel)
}

func onNewLeader(id, current string) {
	if current != id {
		logrus.Infof("New leader observed: %s", current)
	}
}
