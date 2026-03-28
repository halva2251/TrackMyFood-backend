package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ChatAsisstant interface {
	Ask(ctx context.Context, scanData *domain.ScanResponse, question string) (string, error)
}

type ChatHandler struct {
	scanRepo ScanLookup
	ai       ChatAsisstant
}

func NewChatHandler(scanRepo ScanLookup, ai ChatAsisstant) *ChatHandler {
	return &ChatHandler{scanRepo: scanRepo, ai: ai}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	barcode := chi.URLParam(r, "barcode")
	if barcode == "" {
		WriteError(w, http.StatusBadRequest, "barcode is required")
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Question == "" {
		WriteError(w, http.StatusBadRequest, "question is required")
		return
	}

	// Fetch full scan data for context
	scanData, err := h.scanRepo.LookupByBarcode(r.Context(), barcode, req.Lot)
	if err != nil {
		WriteError(w, http.StatusNotFound, "could not find product data for context")
		return
	}

	// Ask the AI
	answer, err := h.ai.Ask(r.Context(), scanData, req.Question)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "AI assistant failed: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, domain.ChatResponse{Answer: answer})
}
