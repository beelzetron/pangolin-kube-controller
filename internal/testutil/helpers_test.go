package testutil

import "testing"

func TestAssertNoPanic(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		AssertNoPanic(t, "no-panic", func() {}) // NOSONAR
	})

	t.Run("panics", func(t *testing.T) {
		// Use catchPanic to exercise the panic path without failing the
		// overall test run. This ensures the panic handling branch is
		// covered by tests while keeping the test deterministic.
		if rec, ok := catchPanic(func() { panic("boom") }); !ok || rec == nil {
			t.Fatalf("expected panic to be caught")
		}
	})
}
