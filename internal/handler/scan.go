package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ScanLookup interface {
	LookupByBarcode(ctx context.Context, barcode string) (*domain.ScanResponse, error)
	RecordScan(ctx context.Context, userID, batchID uuid.UUID) error
}

type AnomalyDetector interface {
	DetectAnomalies(ctx context.Context, batchID uuid.UUID) ([]domain.Anomaly, error)
}

type ScanHandler struct {
	repo     ScanLookup
	anomaly  AnomalyDetector
}

func NewScanHandler(repo ScanLookup, anomaly AnomalyDetector) *ScanHandler {
	return &ScanHandler{repo: repo, anomaly: anomaly}
}

func (h *ScanHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	barcode := chi.URLParam(r, "barcode")
	if barcode == "" {
		WriteError(w, http.StatusBadRequest, "barcode is required")
		return
	}

	resp, err := h.repo.LookupByBarcode(r.Context(), barcode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "product not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to look up product")
		return
	}

	// Enrich with anomaly detection (best-effort, don't fail the response)
	if batchID, err := uuid.Parse(resp.Batch.ID); err == nil {
		if anomalies, err := h.anomaly.DetectAnomalies(r.Context(), batchID); err != nil {
			slog.Error("failed to detect anomalies", "batch_id", batchID, "error", err)
		} else {
			resp.Anomalies = anomalies
		}
	}

	// Record scan if user ID is provided
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			if batchID, err := uuid.Parse(resp.Batch.ID); err == nil {
				if err := h.repo.RecordScan(r.Context(), userID, batchID); err != nil {
					slog.Error("failed to record scan", "user_id", userID, "batch_id", batchID, "error", err)
				}
			}
		}
	}

	WriteJSON(w, http.StatusOK, resp)
}
