package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type RecallCreator interface {
	Create(ctx context.Context, recall domain.Recall) (domain.Recall, error)
	ZeroBatchScore(ctx context.Context, batchID uuid.UUID) error
	GetAffectedUsers(ctx context.Context, batchID uuid.UUID) ([]domain.User, error)
}

type RecallHandler struct {
	repo RecallCreator
}

func NewRecallHandler(repo RecallCreator) *RecallHandler {
	return &RecallHandler{repo: repo}
}

type CreateRecallRequest struct {
	BatchID      string `json:"batch_id"`
	Severity     string `json:"severity"`
	Reason       string `json:"reason"`
	Instructions string `json:"instructions"`
}

type CreateRecallResponse struct {
	Recall        domain.Recall `json:"recall"`
	AffectedUsers []domain.User `json:"affected_users"`
}

func (h *RecallHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var req CreateRecallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	batchID, err := uuid.Parse(req.BatchID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch_id")
		return
	}

	validSeverities := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	if !validSeverities[req.Severity] {
		WriteError(w, http.StatusBadRequest, "invalid severity")
		return
	}

	if req.Reason == "" || req.Instructions == "" {
		WriteError(w, http.StatusBadRequest, "reason and instructions are required")
		return
	}

	recall := domain.Recall{
		BatchID:      batchID,
		Severity:     req.Severity,
		Reason:       req.Reason,
		Instructions: req.Instructions,
	}

	created, err := h.repo.Create(r.Context(), recall)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create recall")
		return
	}

	// Zero the batch trust score
	if err := h.repo.ZeroBatchScore(r.Context(), batchID); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to update batch score")
		return
	}

	// Get affected users (those who scanned this batch)
	affected, err := h.repo.GetAffectedUsers(r.Context(), batchID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get affected users")
		return
	}
	if affected == nil {
		affected = []domain.User{}
	}

	WriteJSON(w, http.StatusCreated, CreateRecallResponse{
		Recall:        created,
		AffectedUsers: affected,
	})
}
