package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
	appmiddleware "github.com/halva2251/trackmyfood-backend/internal/middleware"
	"github.com/halva2251/trackmyfood-backend/internal/repository"
	"github.com/halva2251/trackmyfood-backend/internal/service"
)

type mockUserStore struct {
	createFunc             func(ctx context.Context, email, displayName, passwordHash string) (*domain.User, error)
	findByEmailFunc        func(ctx context.Context, email string) (*repository.UserWithHash, error)
	findByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	getScanHistoryFunc     func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.ScanHistoryEntry, int, error)
	deleteScanHistoryFunc  func(ctx context.Context, userID, entryID uuid.UUID) error
}

func (m *mockUserStore) Create(ctx context.Context, email, displayName, passwordHash string) (*domain.User, error) {
	return m.createFunc(ctx, email, displayName, passwordHash)
}

func (m *mockUserStore) FindByEmail(ctx context.Context, email string) (*repository.UserWithHash, error) {
	return m.findByEmailFunc(ctx, email)
}

func (m *mockUserStore) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return m.findByIDFunc(ctx, id)
}

func (m *mockUserStore) GetScanHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.ScanHistoryEntry, int, error) {
	if m.getScanHistoryFunc != nil {
		return m.getScanHistoryFunc(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockUserStore) DeleteScanHistoryEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	if m.deleteScanHistoryFunc != nil {
		return m.deleteScanHistoryFunc(ctx, userID, entryID)
	}
	return nil
}

type mockTokenGen struct {
	generateFunc func(userID uuid.UUID) (service.TokenPair, error)
	refreshFunc  func(token string) (uuid.UUID, error)
}

func (m *mockTokenGen) GenerateTokenPair(userID uuid.UUID) (service.TokenPair, error) {
	return m.generateFunc(userID)
}

func (m *mockTokenGen) ValidateRefreshToken(token string) (uuid.UUID, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(token)
	}
	return uuid.Nil, pgx.ErrNoRows
}

func newAuthRouter(h *handler.AuthHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/auth/register", h.Register)
	r.Post("/api/auth/login", h.Login)
	r.Post("/api/auth/refresh", h.Refresh)
	return r
}

func TestAuthHandler_Register_Success(t *testing.T) {
	userID := uuid.New()

	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return nil, pgx.ErrNoRows
		},
		createFunc: func(_ context.Context, email, displayName, _ string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email, DisplayName: &displayName}, nil
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(uid uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 123}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":        "test@example.com",
		"password":     "password123",
		"display_name": "Test User",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return &repository.UserWithHash{}, nil // email exists
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":        "existing@example.com",
		"password":     "password123",
		"display_name": "Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestAuthHandler_Register_Validation(t *testing.T) {
	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return nil, pgx.ErrNoRows
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}
	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	tests := []struct {
		name    string
		body    map[string]string
		wantErr string
	}{
		{"missing email", map[string]string{"password": "12345678", "display_name": "T"}, "valid email is required"},
		{"invalid email", map[string]string{"email": "nope", "password": "12345678", "display_name": "T"}, "valid email is required"},
		{"short password", map[string]string{"email": "user@example.com", "password": "1234567", "display_name": "T"}, "password must be at least 8 characters"},
		{"missing name", map[string]string{"email": "user@example.com", "password": "12345678"}, "display name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			var resp handler.Response
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Error != tt.wantErr {
				t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
			}
		})
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	userID := uuid.New()
	hash, _ := service.HashPassword("correct-password")

	displayName := "Demo"
	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return &repository.UserWithHash{
				User:         domain.User{ID: userID, Email: "test@example.com", DisplayName: &displayName},
				PasswordHash: hash,
			}, nil
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(uid uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresAt: 123}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "correct-password"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	hash, _ := service.HashPassword("correct-password")

	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return &repository.UserWithHash{PasswordHash: hash}, nil
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "wrong-password"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	users := &mockUserStore{
		findByEmailFunc: func(_ context.Context, _ string) (*repository.UserWithHash, error) {
			return nil, pgx.ErrNoRows
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "password"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Me(t *testing.T) {
	userID := uuid.New()
	displayName := "Test User"

	users := &mockUserStore{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			if id == userID {
				return &domain.User{ID: userID, Email: "test@example.com", DisplayName: &displayName}, nil
			}
			return nil, pgx.ErrNoRows
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)

	r := chi.NewRouter()
	r.Get("/api/user/me", h.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	ctx := context.WithValue(req.Context(), appmiddleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	users := &mockUserStore{}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)

	r := chi.NewRouter()
	r.Get("/api/user/me", h.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	userID := uuid.New()

	users := &mockUserStore{
		findByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: id}, nil
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{AccessToken: "new-at", RefreshToken: "new-rt", ExpiresAt: 999}, nil
		},
		refreshFunc: func(_ string) (uuid.UUID, error) {
			return userID, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)
	r := newAuthRouter(h)

	body, _ := json.Marshal(map[string]string{"refresh_token": "valid-refresh"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestAuthHandler_ScanHistory(t *testing.T) {
	userID := uuid.New()
	batchID := uuid.New()

	users := &mockUserStore{
		getScanHistoryFunc: func(_ context.Context, uid uuid.UUID, limit, offset int) ([]repository.ScanHistoryEntry, int, error) {
			return []repository.ScanHistoryEntry{
				{ID: uuid.New(), ProductName: "Strawberries", Barcode: "7610000000001", BatchID: batchID},
			}, 1, nil
		},
	}
	tokens := &mockTokenGen{
		generateFunc: func(_ uuid.UUID) (service.TokenPair, error) {
			return service.TokenPair{}, nil
		},
	}

	h := handler.NewAuthHandler(users, tokens)

	r := chi.NewRouter()
	r.Get("/api/user/scan-history", h.ScanHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/user/scan-history", nil)
	ctx := context.WithValue(req.Context(), appmiddleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Data struct {
			Scans []any `json:"scans"`
			Total int   `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data.Scans) != 1 {
		t.Errorf("scans len = %d, want 1", len(resp.Data.Scans))
	}
	if resp.Data.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Data.Total)
	}
}
