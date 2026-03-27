package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

// AlternativesLookup defines the query needed to find alternative products.
type AlternativesLookup interface {
	GetAlternatives(ctx context.Context, productID uuid.UUID, category string, minScore float64) ([]domain.ScanProduct, error)
}

// AlternativesHandler serves GET /api/scan/{barcode}/alternatives.
type AlternativesHandler struct {
	scan         ScanLookup
	alternatives AlternativesLookup
}

func NewAlternativesHandler(scan ScanLookup, alternatives AlternativesLookup) *AlternativesHandler {
	return &AlternativesHandler{scan: scan, alternatives: alternatives}
}

// GetAlternatives returns products in the same category with a higher trust score
// than the scanned product's current best batch score.
func (h *AlternativesHandler) GetAlternatives(w http.ResponseWriter, r *http.Request) {
	barcode := chi.URLParam(r, "barcode")
	if barcode == "" {
		WriteError(w, http.StatusBadRequest, "barcode is required")
		return
	}

	if !isValidBarcode(barcode) {
		WriteError(w, http.StatusBadRequest, "invalid barcode format")
		return
	}

	scanResp, err := h.scan.LookupByBarcode(r.Context(), barcode, "")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "product not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "failed to look up product")
		return
	}

	productID, err := uuid.Parse(scanResp.Product.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "invalid product id in database")
		return
	}

	alts, err := h.alternatives.GetAlternatives(
		r.Context(),
		productID,
		scanResp.Product.Category,
		scanResp.TrustScore.Overall,
	)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get alternatives")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"alternatives": alts})
}
