package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivezHandler(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/livez", http.NoBody)
	rec := httptest.NewRecorder()

	livezHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("livezHandler() status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("livezHandler() body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns 200 when readiness is true", func(t *testing.T) {
		t.Parallel()
		handler := readyHandler(func() bool { return true })
		req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("readyHandler() status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("readyHandler() body = %q, want %q", rec.Body.String(), "ok")
		}
	})

	t.Run("returns 503 when readiness is false", func(t *testing.T) {
		t.Parallel()
		handler := readyHandler(func() bool { return false })
		req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("readyHandler() status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if rec.Body.String() != "not ready" {
			t.Errorf("readyHandler() body = %q, want %q", rec.Body.String(), "not ready")
		}
	})
}
