package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halva2251/trackmyfood-backend/internal/middleware"
)

func TestAdminAuth(t *testing.T) {
	apiKey := "test-key"
	mw := middleware.AdminAuth(apiKey)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid api key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("invalid api key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		rr := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})

	t.Run("no api key set (dev mode)", func(t *testing.T) {
		mwDev := middleware.AdminAuth("")
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		mwDev(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}
