package testutil

import "testing"

// AssertNoPanic runs fn and fails the test if it panics. The provided name
// is included in the failure message to aid debugging. This mirrors the
// previous local helper used across orchestration tests.
func AssertNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	if rec, ok := catchPanic(fn); ok {
		t.Fatalf("panic in %s: %v", name, rec)
	}
}

// catchPanic executes fn and returns the recovered value and true if fn panicked.
// This is exposed to tests in the same package so tests can assert panic
// behaviour without causing the test process to fail.
func catchPanic(fn func()) (interface{}, bool) {
	var rec interface{}
	var ok bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				rec = r
				ok = true
			}
		}()
		fn()
	}()
	return rec, ok
}
