package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
)

type mockTemperatureRepo struct {
	getFunc func(ctx context.Context, batchID uuid.UUID) ([]domain.TemperatureReading, error)
}

func (m *mockTemperatureRepo) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]domain.TemperatureReading, error) {
	return m.getFunc(ctx, batchID)
}

func newTempRouter(h *handler.TemperatureHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/batch/{id}/temperature", h.GetByBatch)
	return r
}

func TestTemperatureHandler_GetByBatch(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")
	now := time.Now()

	tests := []struct {
		name       string
		id         string
		mock       *mockTemperatureRepo
		wantStatus int
		wantCount  int
		wantErr    string
	}{
		{
			name: "success with readings",
			id:   batchID.String(),
			mock: &mockTemperatureRepo{
				getFunc: func(_ context.Context, _ uuid.UUID) ([]domain.TemperatureReading, error) {
					return []domain.TemperatureReading{
						{ID: uuid.New(), BatchID: batchID, RecordedAt: now, TempCelsius: 2.1, MinAcceptable: 1.0, MaxAcceptable: 4.0},
						{ID: uuid.New(), BatchID: batchID, RecordedAt: now.Add(time.Hour), TempCelsius: 2.3, MinAcceptable: 1.0, MaxAcceptable: 4.0},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "empty readings returns empty array",
			id:   batchID.String(),
			mock: &mockTemperatureRepo{
				getFunc: func(_ context.Context, _ uuid.UUID) ([]domain.TemperatureReading, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "invalid uuid",
			id:         "not-a-uuid",
			mock:       &mockTemperatureRepo{},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch ID",
		},
		{
			name: "db error",
			id:   batchID.String(),
			mock: &mockTemperatureRepo{
				getFunc: func(_ context.Context, _ uuid.UUID) ([]domain.TemperatureReading, error) {
					return nil, fmt.Errorf("connection refused")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get temperature readings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewTemperatureHandler(tt.mock)
			r := newTempRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/batch/"+tt.id+"/temperature", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				var resp handler.Response
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			} else {
				var resp struct {
					Success bool                       `json:"success"`
					Data    []domain.TemperatureReading `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Data) != tt.wantCount {
					t.Errorf("readings count = %d, want %d", len(resp.Data), tt.wantCount)
				}
			}
		})
	}
}
