package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
)

type mockAlternativesLookup struct {
	getFunc func(ctx context.Context, productID uuid.UUID, category string, minScore float64) ([]domain.ScanProduct, error)
}

func (m *mockAlternativesLookup) GetAlternatives(ctx context.Context, productID uuid.UUID, category string, minScore float64) ([]domain.ScanProduct, error) {
	return m.getFunc(ctx, productID, category, minScore)
}

func newAlternativesRouter(h *handler.AlternativesHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/scan/{barcode}/alternatives", h.GetAlternatives)
	return r
}

func TestAlternativesHandler_GetAlternatives(t *testing.T) {
	productID := uuid.New()
	batchID := uuid.New()

	baseScanResp := &domain.ScanResponse{
		Product: domain.ScanProduct{
			ID:       productID.String(),
			Name:     "Organic Strawberries",
			Category: "fruits",
			Barcode:  "7610000000001",
		},
		Batch: domain.ScanBatch{ID: batchID.String()},
		TrustScore: domain.ScanTrustScore{Overall: 52.0},
	}

	altProduct := domain.ScanProduct{
		ID:       uuid.New().String(),
		Name:     "Bio Blueberries",
		Category: "fruits",
		Barcode:  "7610000000009",
		Producer: domain.ScanProducer{Name: "Bio Farm"},
	}

	tests := []struct {
		name         string
		barcode      string
		scanMock     *mockScanRepo
		altMock      *mockAlternativesLookup
		wantStatus   int
		wantErr      string
		wantAltCount int
	}{
		{
			name:    "success — two alternatives returned",
			barcode: "7610000000001",
			scanMock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _ string) (*domain.ScanResponse, error) {
					return baseScanResp, nil
				},
			},
			altMock: &mockAlternativesLookup{
				getFunc: func(_ context.Context, _ uuid.UUID, _ string, _ float64) ([]domain.ScanProduct, error) {
					return []domain.ScanProduct{altProduct, altProduct}, nil
				},
			},
			wantStatus:   http.StatusOK,
			wantAltCount: 2,
		},
		{
			name:    "success — no alternatives found",
			barcode: "7610000000001",
			scanMock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _ string) (*domain.ScanResponse, error) {
					return baseScanResp, nil
				},
			},
			altMock: &mockAlternativesLookup{
				getFunc: func(_ context.Context, _ uuid.UUID, _ string, _ float64) ([]domain.ScanProduct, error) {
					return []domain.ScanProduct{}, nil
				},
			},
			wantStatus:   http.StatusOK,
			wantAltCount: 0,
		},
		{
			name:    "product not found",
			barcode: "9999999999999",
			scanMock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _ string) (*domain.ScanResponse, error) {
					return nil, pgx.ErrNoRows
				},
			},
			altMock:    &mockAlternativesLookup{},
			wantStatus: http.StatusNotFound,
			wantErr:    "product not found",
		},
		{
			name:    "scan internal error",
			barcode: "7610000000001",
			scanMock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _ string) (*domain.ScanResponse, error) {
					return nil, fmt.Errorf("db failure")
				},
			},
			altMock:    &mockAlternativesLookup{},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to look up product",
		},
		{
			name:    "alternatives query error",
			barcode: "7610000000001",
			scanMock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _ string) (*domain.ScanResponse, error) {
					return baseScanResp, nil
				},
			},
			altMock: &mockAlternativesLookup{
				getFunc: func(_ context.Context, _ uuid.UUID, _ string, _ float64) ([]domain.ScanProduct, error) {
					return nil, fmt.Errorf("query failed")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get alternatives",
		},
		{
			name:       "invalid barcode format",
			barcode:    "abc<script>",
			scanMock:   &mockScanRepo{},
			altMock:    &mockAlternativesLookup{},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid barcode format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewAlternativesHandler(tt.scanMock, tt.altMock)
			r := newAlternativesRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/scan/"+tt.barcode+"/alternatives", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantErr != "" {
				var resp handler.Response
				if jsonErr := json.Unmarshal(w.Body.Bytes(), &resp); jsonErr == nil && resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			}

			if tt.wantAltCount > 0 {
				var resp struct {
					Data struct {
						Alternatives []domain.ScanProduct `json:"alternatives"`
					} `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Data.Alternatives) != tt.wantAltCount {
					t.Errorf("alternatives count = %d, want %d", len(resp.Data.Alternatives), tt.wantAltCount)
				}
			}
		})
	}
}
