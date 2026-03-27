package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ComplaintCreator interface {
	Create(ctx context.Context, c domain.Complaint) (domain.Complaint, error)
}

type ScoreRecalculator interface {
	Recalculate(ctx context.Context, batchID uuid.UUID) error
}

type ComplaintHandler struct {
	repo       ComplaintCreator
	trustScore ScoreRecalculator
	wg         *sync.WaitGroup
}

func NewComplaintHandler(repo ComplaintCreator, ts ScoreRecalculator, wg *sync.WaitGroup) *ComplaintHandler {
	return &ComplaintHandler{repo: repo, trustScore: ts, wg: wg}
}

type CreateComplaintRequest struct {
	BatchID       string  `json:"batch_id"`
	UserID        string  `json:"user_id"`
	ComplaintType string  `json:"complaint_type"`
	Description   *string `json:"description,omitempty"`
	PhotoURL      *string `json:"photo_url,omitempty"`
}

func (h *ComplaintHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var req CreateComplaintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	batchID, err := uuid.Parse(req.BatchID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch_id")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	validTypes := map[string]bool{
		"taste_smell": true, "packaging_damaged": true, "foreign_object": true,
		"suspected_spoilage": true, "other": true,
	}
	if !validTypes[req.ComplaintType] {
		WriteError(w, http.StatusBadRequest, "invalid complaint_type")
		return
	}

	if req.PhotoURL != nil && *req.PhotoURL != "" {
		if _, err := url.ParseRequestURI(*req.PhotoURL); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid photo_url")
			return
		}
		// Must start with http/https
		u, _ := url.Parse(*req.PhotoURL)
		if u.Scheme != "http" && u.Scheme != "https" {
			WriteError(w, http.StatusBadRequest, "photo_url must use http or https")
			return
		}
	}

	complaint := domain.Complaint{
		BatchID:       batchID,
		UserID:        userID,
		ComplaintType: req.ComplaintType,
		Description:   req.Description,
		PhotoURL:      req.PhotoURL,
	}

	created, err := h.repo.Create(r.Context(), complaint)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create complaint")
		return
	}

	// Async trust score recalculation
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.trustScore.Recalculate(ctx, batchID); err != nil {
			slog.Error("failed to recalculate trust score", "batch_id", batchID, "error", err)
		}
	}()

	WriteJSON(w, http.StatusCreated, created)
}
