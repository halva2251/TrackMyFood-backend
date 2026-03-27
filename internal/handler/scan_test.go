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
	appmiddleware "github.com/halva2251/trackmyfood-backend/internal/middleware"
)

type mockScanRepo struct {
	lookupFunc     func(ctx context.Context, barcode, lot string) (*domain.ScanResponse, error)
	recordScanFunc func(ctx context.Context, userID, batchID uuid.UUID) error
}

func (m *mockScanRepo) LookupByBarcode(ctx context.Context, barcode, lot string) (*domain.ScanResponse, error) {
	return m.lookupFunc(ctx, barcode, lot)
}

func (m *mockScanRepo) RecordScan(ctx context.Context, userID, batchID uuid.UUID) error {
	if m.recordScanFunc != nil {
		return m.recordScanFunc(ctx, userID, batchID)
	}
	return nil
}

type mockAnomalyDetector struct{}

func (m *mockAnomalyDetector) DetectAnomalies(_ context.Context, _ uuid.UUID) ([]domain.Anomaly, error) {
	return nil, nil
}

func newScanRouter(h *handler.ScanHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/scan/{barcode}", h.Lookup)
	return r
}

func TestScanHandler_Lookup(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")

	scanResp := &domain.ScanResponse{
		Product: domain.ScanProduct{
			ID:       "00000000-0000-0000-0001-000000000001",
			Name:     "Organic Strawberries 500g",
			Category: "fruits",
			Barcode:  "7610000000001",
			Producer: domain.ScanProducer{
				ID:      "00000000-0000-0000-0000-000000000001",
				Name:    "Bio Hof Thurgau",
				Country: "CH",
			},
		},
		Batch: domain.ScanBatch{
			ID:             batchID.String(),
			LotNumber:      "LOT-2026-0312-A",
			ProductionDate: "2026-03-12T06:00:00Z",
		},
		TrustScore: domain.ScanTrustScore{
			Overall: 94,
			Label:   "Excellent",
			Color:   "green",
		},
		Journey: []domain.ScanJourneyStep{
			{StepOrder: 1, StepType: "harvested", Location: "Bio Hof, Frauenfeld"},
		},
	}

	tests := []struct {
		name       string
		barcode    string
		mock       *mockScanRepo
		wantStatus int
		wantErr    string
	}{
		{
			name:    "success",
			barcode: "7610000000001",
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, barcode, _ string) (*domain.ScanResponse, error) {
					if barcode == "7610000000001" {
						return scanResp, nil
					}
					return nil, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "not found",
			barcode: "9999999999999",
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
					return nil, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "product not found",
		},
		{
			name:    "internal error",
			barcode: "7610000000001",
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
					return nil, fmt.Errorf("db connection failed")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to look up product",
		},
		{
			name:    "barcode too long (101 chars)",
			barcode: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"[:101], // 101 'a'
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid barcode format",
		},
		{
			name:    "barcode with invalid characters",
			barcode: "abc<script>",
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid barcode format",
		},
		{
			name:    "valid alphanumeric QR code",
			barcode: "QR-2026-ABC123",
			mock: &mockScanRepo{
				lookupFunc: func(_ context.Context, barcode, _ string) (*domain.ScanResponse, error) {
					if barcode == "QR-2026-ABC123" {
						return scanResp, nil
					}
					return nil, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewScanHandler(tt.mock, &mockAnomalyDetector{})
			r := newScanRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/scan/"+tt.barcode, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp handler.Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.wantErr != "" {
				if resp.Success {
					t.Error("expected success = false")
				}
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			} else {
				if !resp.Success {
					t.Error("expected success = true")
				}
			}
		})
	}
}

func TestScanHandler_Lookup_RecordsScan(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0004-000000000001")

	var recordedUserID, recordedBatchID uuid.UUID

	mock := &mockScanRepo{
		lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
			return &domain.ScanResponse{
				Batch: domain.ScanBatch{ID: batchID.String()},
			}, nil
		},
		recordScanFunc: func(_ context.Context, uid, bid uuid.UUID) error {
			recordedUserID = uid
			recordedBatchID = bid
			return nil
		},
	}

	h := handler.NewScanHandler(mock, &mockAnomalyDetector{})
	r := newScanRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/scan/7610000000001", nil)
	// Inject user ID into context (normally done by middleware)
	ctx := context.WithValue(req.Context(), appmiddleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if recordedUserID != userID {
		t.Errorf("recorded user = %s, want %s", recordedUserID, userID)
	}
	if recordedBatchID != batchID {
		t.Errorf("recorded batch = %s, want %s", recordedBatchID, batchID)
	}
}

func TestScanHandler_Lookup_SkipsRecordWhenNoUserInContext(t *testing.T) {
	var recorded bool
	mock := &mockScanRepo{
		lookupFunc: func(_ context.Context, _, _ string) (*domain.ScanResponse, error) {
			return &domain.ScanResponse{
				Batch: domain.ScanBatch{ID: uuid.New().String()},
			}, nil
		},
		recordScanFunc: func(_ context.Context, _, _ uuid.UUID) error {
			recorded = true
			return nil
		},
	}

	h := handler.NewScanHandler(mock, &mockAnomalyDetector{})
	r := newScanRouter(h)

	// No user ID in context — scan should not be recorded
	req := httptest.NewRequest(http.MethodGet, "/api/scan/7610000000001", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if recorded {
		t.Error("should not record scan when no user in context")
	}
}
