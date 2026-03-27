package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/middleware"
)

type mockTokenValidator struct {
	validateFunc func(token string) (uuid.UUID, error)
}

func (m *mockTokenValidator) ValidateAccessToken(token string) (uuid.UUID, error) {
	return m.validateFunc(token)
}

func TestUserAuth_ValidToken(t *testing.T) {
	userID := uuid.New()
	validator := &mockTokenValidator{
		validateFunc: func(_ string) (uuid.UUID, error) {
			return userID, nil
		},
	}

	var gotUserID uuid.UUID
	var gotOK bool
	handler := middleware.UserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, gotOK = middleware.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !gotOK {
		t.Error("user ID should be in context")
	}
	if gotUserID != userID {
		t.Errorf("user ID = %s, want %s", gotUserID, userID)
	}
}

func TestUserAuth_MissingHeader(t *testing.T) {
	validator := &mockTokenValidator{}

	handler := middleware.UserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "missing authorization header" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestUserAuth_InvalidFormat(t *testing.T) {
	validator := &mockTokenValidator{}

	handler := middleware.UserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUserAuth_InvalidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFunc: func(_ string) (uuid.UUID, error) {
			return uuid.Nil, http.ErrNoCookie // any error
		},
	}

	handler := middleware.UserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestOptionalUserAuth_WithToken(t *testing.T) {
	userID := uuid.New()
	validator := &mockTokenValidator{
		validateFunc: func(_ string) (uuid.UUID, error) {
			return userID, nil
		},
	}

	var gotUserID uuid.UUID
	var gotOK bool
	handler := middleware.OptionalUserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, gotOK = middleware.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !gotOK || gotUserID != userID {
		t.Error("user ID should be in context from Bearer token")
	}
}

func TestOptionalUserAuth_XUserIDIgnored(t *testing.T) {
	validator := &mockTokenValidator{}

	var gotOK bool
	handler := middleware.OptionalUserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotOK = middleware.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotOK {
		t.Error("X-User-ID header should NOT be accepted as authentication")
	}
}

func TestOptionalUserAuth_NoAuth(t *testing.T) {
	validator := &mockTokenValidator{}

	var gotOK bool
	handler := middleware.OptionalUserAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotOK = middleware.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotOK {
		t.Error("user ID should NOT be in context when no auth provided")
	}
}
