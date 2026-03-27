package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
)

// ---- mocks ----

type mockProducerRepo struct {
	createBatchFunc        func(ctx context.Context, b domain.Batch) (domain.Batch, error)
	addJourneyStepFunc     func(ctx context.Context, s domain.JourneyStep) (domain.JourneyStep, error)
	addTemperatureFunc     func(ctx context.Context, tr domain.TemperatureReading) (domain.TemperatureReading, error)
	addQualityCheckFunc    func(ctx context.Context, qc domain.QualityCheck) (domain.QualityCheck, error)
}

func (m *mockProducerRepo) CreateBatch(ctx context.Context, b domain.Batch) (domain.Batch, error) {
	return m.createBatchFunc(ctx, b)
}
func (m *mockProducerRepo) AddJourneyStep(ctx context.Context, s domain.JourneyStep) (domain.JourneyStep, error) {
	return m.addJourneyStepFunc(ctx, s)
}
func (m *mockProducerRepo) AddTemperatureReading(ctx context.Context, tr domain.TemperatureReading) (domain.TemperatureReading, error) {
	return m.addTemperatureFunc(ctx, tr)
}
func (m *mockProducerRepo) AddQualityCheck(ctx context.Context, qc domain.QualityCheck) (domain.QualityCheck, error) {
	return m.addQualityCheckFunc(ctx, qc)
}

func newProducerRouter(h *handler.ProducerHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/producer/batches", h.CreateBatch)
	r.Post("/api/producer/batches/{id}/journey-steps", h.AddJourneyStep)
	r.Post("/api/producer/batches/{id}/temperature-readings", h.AddTemperatureReading)
	r.Post("/api/producer/batches/{id}/quality-checks", h.AddQualityCheck)
	return r
}

// ---- CreateBatch tests ----

func TestProducerHandler_CreateBatch(t *testing.T) {
	productID := uuid.New()
	batchID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	successRepo := &mockProducerRepo{
		createBatchFunc: func(_ context.Context, b domain.Batch) (domain.Batch, error) {
			b.ID = batchID
			b.CreatedAt = now
			return b, nil
		},
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"LOT-001","production_date":"2026-03-12T06:00:00Z"}`, productID),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "with expiry date",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"LOT-002","production_date":"2026-03-12T06:00:00Z","expiry_date":"2026-06-12T06:00:00Z"}`, productID),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "invalid product_id",
			body:       `{"product_id":"not-a-uuid","lot_number":"LOT-001","production_date":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid product_id: must be a valid UUID",
		},
		{
			name:       "empty lot_number",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"","production_date":"2026-03-12T06:00:00Z"}`, productID),
			wantStatus: http.StatusBadRequest,
			wantErr:    "lot_number is required",
		},
		{
			name:       "invalid production_date",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"LOT-001","production_date":"not-a-date"}`, productID),
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid production_date: must be RFC3339",
		},
		{
			name:       "invalid expiry_date",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"LOT-001","production_date":"2026-03-12T06:00:00Z","expiry_date":"bad"}`, productID),
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid expiry_date: must be RFC3339",
		},
		{
			name:       "repo error",
			body:       fmt.Sprintf(`{"product_id":%q,"lot_number":"LOT-001","production_date":"2026-03-12T06:00:00Z"}`, productID),
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to create batch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := successRepo
			if tt.wantStatus == http.StatusInternalServerError {
				repo = &mockProducerRepo{
					createBatchFunc: func(_ context.Context, _ domain.Batch) (domain.Batch, error) {
						return domain.Batch{}, fmt.Errorf("db error")
					},
				}
			}
			h := handler.NewProducerHandler(repo, &mockScoreRecalculator{}, &sync.WaitGroup{})
			r := newProducerRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/api/producer/batches", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantErr != "" {
				var resp handler.Response
				json.Unmarshal(w.Body.Bytes(), &resp) //nolint
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			}
		})
	}
}

// ---- AddJourneyStep tests ----

func TestProducerHandler_AddJourneyStep(t *testing.T) {
	batchID := uuid.New()
	stepID := uuid.New()

	successRepo := &mockProducerRepo{
		addJourneyStepFunc: func(_ context.Context, s domain.JourneyStep) (domain.JourneyStep, error) {
			s.ID = stepID
			return s, nil
		},
	}

	tests := []struct {
		name       string
		batchID    string
		body       string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success - harvested",
			batchID:    batchID.String(),
			body:       `{"step_order":1,"step_type":"harvested","location":"Farm A","arrived_at":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "success - transported with departed_at",
			batchID:    batchID.String(),
			body:       `{"step_order":2,"step_type":"transported","location":"Truck","arrived_at":"2026-03-12T08:00:00Z","departed_at":"2026-03-12T10:00:00Z"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid batch id",
			batchID:    "not-a-uuid",
			body:       `{"step_order":1,"step_type":"harvested","location":"Farm","arrived_at":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch id: must be a valid UUID",
		},
		{
			name:       "invalid json",
			batchID:    batchID.String(),
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "invalid step_type",
			batchID:    batchID.String(),
			body:       `{"step_order":1,"step_type":"flying","location":"Farm","arrived_at":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid step_type: must be one of harvested, processed, stored, transported, delivered",
		},
		{
			name:       "empty location",
			batchID:    batchID.String(),
			body:       `{"step_order":1,"step_type":"harvested","location":"","arrived_at":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "location is required",
		},
		{
			name:       "invalid arrived_at",
			batchID:    batchID.String(),
			body:       `{"step_order":1,"step_type":"harvested","location":"Farm","arrived_at":"not-a-date"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid arrived_at: must be RFC3339",
		},
		{
			name:    "repo error",
			batchID: batchID.String(),
			body:    `{"step_order":1,"step_type":"harvested","location":"Farm","arrived_at":"2026-03-12T06:00:00Z"}`,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to add journey step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := successRepo
			if tt.wantStatus == http.StatusInternalServerError {
				repo = &mockProducerRepo{
					addJourneyStepFunc: func(_ context.Context, _ domain.JourneyStep) (domain.JourneyStep, error) {
						return domain.JourneyStep{}, fmt.Errorf("db error")
					},
				}
			}
			h := handler.NewProducerHandler(repo, &mockScoreRecalculator{}, &sync.WaitGroup{})
			r := newProducerRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/api/producer/batches/"+tt.batchID+"/journey-steps", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantErr != "" {
				var resp handler.Response
				json.Unmarshal(w.Body.Bytes(), &resp) //nolint
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			}
		})
	}
}

// ---- AddTemperatureReading tests ----

func TestProducerHandler_AddTemperatureReading(t *testing.T) {
	batchID := uuid.New()
	readingID := uuid.New()

	successRepo := &mockProducerRepo{
		addTemperatureFunc: func(_ context.Context, tr domain.TemperatureReading) (domain.TemperatureReading, error) {
			tr.ID = readingID
			return tr, nil
		},
	}

	tests := []struct {
		name       string
		batchID    string
		body       string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success",
			batchID:    batchID.String(),
			body:       `{"recorded_at":"2026-03-12T08:00:00Z","temp_celsius":3.5,"min_acceptable":0.0,"max_acceptable":4.0}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid batch id",
			batchID:    "bad",
			body:       `{"recorded_at":"2026-03-12T08:00:00Z","temp_celsius":3.5,"min_acceptable":0.0,"max_acceptable":4.0}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch id: must be a valid UUID",
		},
		{
			name:       "invalid recorded_at",
			batchID:    batchID.String(),
			body:       `{"recorded_at":"not-a-date","temp_celsius":3.5,"min_acceptable":0.0,"max_acceptable":4.0}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid recorded_at: must be RFC3339",
		},
		{
			name:       "min >= max",
			batchID:    batchID.String(),
			body:       `{"recorded_at":"2026-03-12T08:00:00Z","temp_celsius":3.5,"min_acceptable":5.0,"max_acceptable":4.0}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "min_acceptable must be less than max_acceptable",
		},
		{
			name:    "repo error",
			batchID: batchID.String(),
			body:    `{"recorded_at":"2026-03-12T08:00:00Z","temp_celsius":3.5,"min_acceptable":0.0,"max_acceptable":4.0}`,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to add temperature reading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := successRepo
			if tt.wantStatus == http.StatusInternalServerError {
				repo = &mockProducerRepo{
					addTemperatureFunc: func(_ context.Context, _ domain.TemperatureReading) (domain.TemperatureReading, error) {
						return domain.TemperatureReading{}, fmt.Errorf("db error")
					},
				}
			}
			h := handler.NewProducerHandler(repo, &mockScoreRecalculator{}, &sync.WaitGroup{})
			r := newProducerRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/api/producer/batches/"+tt.batchID+"/temperature-readings", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantErr != "" {
				var resp handler.Response
				json.Unmarshal(w.Body.Bytes(), &resp) //nolint
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			}
		})
	}
}

// ---- AddQualityCheck tests ----

func TestProducerHandler_AddQualityCheck(t *testing.T) {
	batchID := uuid.New()
	checkID := uuid.New()

	successRepo := &mockProducerRepo{
		addQualityCheckFunc: func(_ context.Context, qc domain.QualityCheck) (domain.QualityCheck, error) {
			qc.ID = checkID
			return qc, nil
		},
	}

	tests := []struct {
		name       string
		batchID    string
		body       string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success - passed",
			batchID:    batchID.String(),
			body:       `{"check_type":"visual","passed":true,"checked_at":"2026-03-12T09:00:00Z"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "success - failed with inspector",
			batchID:    batchID.String(),
			body:       `{"check_type":"microbiological","passed":false,"checked_at":"2026-03-12T09:00:00Z","inspector":"Dr. Müller"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid batch id",
			batchID:    "bad",
			body:       `{"check_type":"visual","passed":true,"checked_at":"2026-03-12T09:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch id: must be a valid UUID",
		},
		{
			name:       "empty check_type",
			batchID:    batchID.String(),
			body:       `{"check_type":"","passed":true,"checked_at":"2026-03-12T09:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "check_type is required",
		},
		{
			name:       "invalid checked_at",
			batchID:    batchID.String(),
			body:       `{"check_type":"visual","passed":true,"checked_at":"not-a-date"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid checked_at: must be RFC3339",
		},
		{
			name:    "repo error",
			batchID: batchID.String(),
			body:    `{"check_type":"visual","passed":true,"checked_at":"2026-03-12T09:00:00Z"}`,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to add quality check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := successRepo
			if tt.wantStatus == http.StatusInternalServerError {
				repo = &mockProducerRepo{
					addQualityCheckFunc: func(_ context.Context, _ domain.QualityCheck) (domain.QualityCheck, error) {
						return domain.QualityCheck{}, fmt.Errorf("db error")
					},
				}
			}
			h := handler.NewProducerHandler(repo, &mockScoreRecalculator{}, &sync.WaitGroup{})
			r := newProducerRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/api/producer/batches/"+tt.batchID+"/quality-checks", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantErr != "" {
				var resp handler.Response
				json.Unmarshal(w.Body.Bytes(), &resp) //nolint
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			}
		})
	}
}
