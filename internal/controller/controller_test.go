package controller

import (
	"testing"

	"pangolin-kube-controller/internal/config"
)

func TestKindForAndTitleCase(t *testing.T) {
	if got := kindFor("middlewares"); got != "Middleware" {
		t.Fatalf("kindFor middlewares = %q", got)
	}
	if got := kindFor("widgets"); got != "Widgets" {
		t.Fatalf("fallback kindFor = %q", got)
	}
	if got := titleCaseFirst(""); got != "" {
		t.Fatalf("titleCaseFirst empty = %q", got)
	}
	if got := titleCaseFirst("x"); got != "X" {
		t.Fatalf("titleCaseFirst = %q", got)
	}
}

func TestIgnoreFieldValidation(t *testing.T) {
	if !ignoreFieldValidation("TraefikService") {
		t.Fatalf("expected TraefikService to be ignored")
	}
	if ignoreFieldValidation("Middleware") {
		t.Fatalf("expected Middleware to not be ignored")
	}
}

func TestExitRequested(t *testing.T) {
	c := &Controller{}
	if c.ExitRequested() {
		t.Fatalf("expected exit not requested initially")
	}
	c.exitRequested.Store(true)
	if !c.ExitRequested() {
		t.Fatalf("expected exit requested after setting")
	}
}

func TestLeaderIdentity(t *testing.T) {
	c := NewController(&config.Config{}, nil, nil, nil)
	id := c.makeLeaderIdentity()
	if id == "" {
		t.Fatalf("expected non-empty leader identity")
	}
}
