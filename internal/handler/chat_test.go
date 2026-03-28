package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

// MockChatAssistant for testing
type mockChat struct {
	answer string
	err    error
}

func (m *mockChat) Ask(ctx context.Context, scanData *domain.ScanResponse, question string) (string, error) {
	return m.answer, m.err
}

// MockScanLookup for providing context to the chat
type mockScanRepo struct {
	resp *domain.ScanResponse
	err  error
}

func (m *mockScanRepo) LookupByBarcode(ctx context.Context, barcode, lot string) (*domain.ScanResponse, error) {
	return m.resp, m.err
}
func (m *mockScanRepo) RecordScan(ctx context.Context, userID, batchID uuid.UUID) error { return nil }

func TestChatHandler_Chat(t *testing.T) {
	mockRepo := &mockScanRepo{
		resp: &domain.ScanResponse{
			Product:    domain.ScanProduct{Name: "Test Product"},
			TrustScore: domain.ScanTrustScore{Overall: 95},
		},
	}
	mockAI := &mockChat{answer: "It is safe to drink."}
	h := NewChatHandler(mockRepo, mockAI)

	t.Run("successful chat", func(t *testing.T) {
		reqBody, _ := json.Marshal(domain.ChatRequest{
			Question: "Is this safe?",
			Lot:      "L123",
		})
		req := httptest.NewRequest("POST", "/api/scan/12345/chat", bytes.NewBuffer(reqBody))
		
		// Add chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("barcode", "12345")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.Chat(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var fullResp Response
		json.NewDecoder(w.Body).Decode(&fullResp)
		
		// Map the Data field back to ChatResponse
		dataJSON, _ := json.Marshal(fullResp.Data)
		var chatResp domain.ChatResponse
		json.Unmarshal(dataJSON, &chatResp)

		if chatResp.Answer != "It is safe to drink." {
			t.Errorf("expected answer 'It is safe to drink.', got '%s'", chatResp.Answer)
		}
	})

	t.Run("missing question", func(t *testing.T) {
		reqBody, _ := json.Marshal(domain.ChatRequest{Question: ""})
		req := httptest.NewRequest("POST", "/api/scan/12345/chat", bytes.NewBuffer(reqBody))
		
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("barcode", "12345")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.Chat(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}
