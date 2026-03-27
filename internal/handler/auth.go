package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/middleware"
	"github.com/halva2251/trackmyfood-backend/internal/repository"
	"github.com/halva2251/trackmyfood-backend/internal/service"
)

type UserStore interface {
	Create(ctx context.Context, email, displayName, passwordHash string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*repository.UserWithHash, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetScanHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.ScanHistoryEntry, int, error)
	DeleteScanHistoryEntry(ctx context.Context, userID, entryID uuid.UUID) error
}

type TokenGenerator interface {
	GenerateTokenPair(userID uuid.UUID) (service.TokenPair, error)
	ValidateRefreshToken(token string) (uuid.UUID, error)
}

type AuthHandler struct {
	users  UserStore
	tokens TokenGenerator
}

func NewAuthHandler(users UserStore, tokens TokenGenerator) *AuthHandler {
	return &AuthHandler{users: users, tokens: tokens}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if _, err := mail.ParseAddress(req.Email); err != nil {
		WriteError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Password) < 8 {
		WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.DisplayName == "" {
		WriteError(w, http.StatusBadRequest, "display name is required")
		return
	}

	// Check if email already exists
	if _, err := h.users.FindByEmail(r.Context(), req.Email); err == nil {
		WriteError(w, http.StatusConflict, "email already registered")
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusInternalServerError, "failed to check email availability")
		return
	}

	hash, err := service.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to process registration")
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, req.DisplayName, hash)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	tokens, err := h.tokens.GenerateTokenPair(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" {
		WriteError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		WriteError(w, http.StatusBadRequest, "password is required")
		return
	}

	user, err := h.users.FindByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	if !service.CheckPassword(user.PasswordHash, req.Password) {
		WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	tokens, err := h.tokens.GenerateTokenPair(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"user":   &user.User,
		"tokens": tokens,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		WriteError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	userID, err := h.tokens.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Verify user still exists
	if _, err := h.users.FindByID(r.Context(), userID); err != nil {
		WriteError(w, http.StatusUnauthorized, "user not found")
		return
	}

	tokens, err := h.tokens.GenerateTokenPair(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	WriteJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.users.FindByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ScanHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, total, err := h.users.GetScanHistory(r.Context(), userID, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get scan history")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"scans": entries,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

func (h *AuthHandler) DeleteScanHistoryEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	entryIDStr := r.URL.Query().Get("id")
	if entryIDStr == "" {
		WriteError(w, http.StatusBadRequest, "scan history entry id is required")
		return
	}
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	if err := h.users.DeleteScanHistoryEntry(r.Context(), userID, entryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "scan history entry not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to delete scan history entry")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
