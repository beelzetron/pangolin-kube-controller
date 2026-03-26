package main

import (
	"context"
	"errors"
	"testing"

	"pangolin-kube-controller/internal/config"
)

func TestMainSuccessExitZero(t *testing.T) {
	oldRun := runController
	oldExit := osExit
	runController = func(_ context.Context, _ *config.Config) error { return nil }
	code := -1
	osExit = func(c int) { code = c }
	t.Cleanup(func() { runController = oldRun; osExit = oldExit })

	main()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestMainErrorExitOne(t *testing.T) {
	oldRun := runController
	oldExit := osExit
	runController = func(_ context.Context, _ *config.Config) error { return errors.New("boom") }
	code := -1
	osExit = func(c int) { code = c }
	t.Cleanup(func() { runController = oldRun; osExit = oldExit })

	main()
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}
