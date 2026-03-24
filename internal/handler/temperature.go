package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type TemperatureGetter interface {
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]domain.TemperatureReading, error)
}

type TemperatureHandler struct {
	repo TemperatureGetter
}

func NewTemperatureHandler(repo TemperatureGetter) *TemperatureHandler {
	return &TemperatureHandler{repo: repo}
}

func (h *TemperatureHandler) GetByBatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch ID")
		return
	}

	readings, err := h.repo.GetByBatchID(r.Context(), batchID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get temperature readings")
		return
	}

	if readings == nil {
		readings = []domain.TemperatureReading{}
	}

	WriteJSON(w, http.StatusOK, readings)
}
