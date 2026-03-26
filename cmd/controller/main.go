// The main package for the Pangolin Kube Controller application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"pangolin-kube-controller/internal/config"
	"pangolin-kube-controller/internal/orchestration"
	"pangolin-kube-controller/internal/version"
)

// seams for testability
var (
	osExit        = os.Exit
	runController = orchestration.Run
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg := config.LoadFromEnv()

	// Early logging of embedded build information (set via -ldflags)
	logrus.Infof("starting pangolin-kube-controller Version=%s Commit=%s Date=%s", version.Version, version.Commit, version.Date)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Exit code managed via deferred function to ensure other defers (like cancel()) run.
	exitCode := 0
	defer func() {
		// ensure logs flushed if needed (logrus doesn't require explicit flush by default)
		osExit(exitCode)
	}()

	if err := runController(ctx, cfg); err != nil {
		logrus.Errorf("application error: %v", err)
		exitCode = 1
		return
	}
}
