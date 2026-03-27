package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

// ProducerRepo defines the write operations used by ProducerHandler.
type ProducerRepo interface {
	CreateBatch(ctx context.Context, b domain.Batch) (domain.Batch, error)
	AddJourneyStep(ctx context.Context, s domain.JourneyStep) (domain.JourneyStep, error)
	AddTemperatureReading(ctx context.Context, tr domain.TemperatureReading) (domain.TemperatureReading, error)
	AddQualityCheck(ctx context.Context, qc domain.QualityCheck) (domain.QualityCheck, error)
}

// ProducerHandler handles supply chain data ingestion endpoints.
type ProducerHandler struct {
	repo       ProducerRepo
	trustScore ScoreRecalculator
	wg         *sync.WaitGroup
}

func NewProducerHandler(repo ProducerRepo, ts ScoreRecalculator, wg *sync.WaitGroup) *ProducerHandler {
	return &ProducerHandler{repo: repo, trustScore: ts, wg: wg}
}

// validStepTypes contains the allowed journey step types.
var validStepTypes = map[string]bool{
	"harvested":   true,
	"processed":   true,
	"stored":      true,
	"transported": true,
	"delivered":   true,
}

// asyncRecalculate triggers an async trust score recalculation for the given batch.
func (h *ProducerHandler) asyncRecalculate(batchID uuid.UUID) {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.trustScore.Recalculate(ctx, batchID); err != nil {
			slog.Error("failed to recalculate trust score", "batch_id", batchID, "error", err)
		}
	}()
}

// CreateBatch handles POST /api/producer/batches.
func (h *ProducerHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		ProductID      string  `json:"product_id"`
		LotNumber      string  `json:"lot_number"`
		ProductionDate string  `json:"production_date"`
		ExpiryDate     *string `json:"expiry_date,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid product_id: must be a valid UUID")
		return
	}

	if req.LotNumber == "" {
		WriteError(w, http.StatusBadRequest, "lot_number is required")
		return
	}

	productionDate, err := time.Parse(time.RFC3339, req.ProductionDate)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid production_date: must be RFC3339")
		return
	}

	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiryDate)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid expiry_date: must be RFC3339")
			return
		}
		expiryDate = &t
	}

	batch := domain.Batch{
		ProductID:      productID,
		LotNumber:      req.LotNumber,
		ProductionDate: productionDate,
		ExpiryDate:     expiryDate,
	}

	created, err := h.repo.CreateBatch(r.Context(), batch)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create batch")
		return
	}

	h.asyncRecalculate(created.ID)

	WriteJSON(w, http.StatusCreated, created)
}

// AddJourneyStep handles POST /api/producer/batches/{id}/journey-steps.
func (h *ProducerHandler) AddJourneyStep(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	batchID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch id: must be a valid UUID")
		return
	}

	var req struct {
		StepOrder  int      `json:"step_order"`
		StepType   string   `json:"step_type"`
		Location   string   `json:"location"`
		Latitude   *float64 `json:"latitude,omitempty"`
		Longitude  *float64 `json:"longitude,omitempty"`
		ArrivedAt  string   `json:"arrived_at"`
		DepartedAt *string  `json:"departed_at,omitempty"`
		Notes      *string  `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validStepTypes[req.StepType] {
		WriteError(w, http.StatusBadRequest, "invalid step_type: must be one of harvested, processed, stored, transported, delivered")
		return
	}

	if req.Location == "" {
		WriteError(w, http.StatusBadRequest, "location is required")
		return
	}

	arrivedAt, err := time.Parse(time.RFC3339, req.ArrivedAt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid arrived_at: must be RFC3339")
		return
	}

	var departedAt *time.Time
	if req.DepartedAt != nil && *req.DepartedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.DepartedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid departed_at: must be RFC3339")
			return
		}
		departedAt = &t
	}

	step := domain.JourneyStep{
		BatchID:    batchID,
		StepOrder:  req.StepOrder,
		StepType:   req.StepType,
		Location:   req.Location,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		ArrivedAt:  arrivedAt,
		DepartedAt: departedAt,
		Notes:      req.Notes,
	}

	created, err := h.repo.AddJourneyStep(r.Context(), step)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to add journey step")
		return
	}

	WriteJSON(w, http.StatusCreated, created)
}

// AddTemperatureReading handles POST /api/producer/batches/{id}/temperature-readings.
func (h *ProducerHandler) AddTemperatureReading(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	batchID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch id: must be a valid UUID")
		return
	}

	var req struct {
		RecordedAt    string   `json:"recorded_at"`
		TempCelsius   float64  `json:"temp_celsius"`
		MinAcceptable float64  `json:"min_acceptable"`
		MaxAcceptable float64  `json:"max_acceptable"`
		Location      *string  `json:"location,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recordedAt, err := time.Parse(time.RFC3339, req.RecordedAt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid recorded_at: must be RFC3339")
		return
	}

	if req.MinAcceptable >= req.MaxAcceptable {
		WriteError(w, http.StatusBadRequest, "min_acceptable must be less than max_acceptable")
		return
	}

	reading := domain.TemperatureReading{
		BatchID:       batchID,
		RecordedAt:    recordedAt,
		TempCelsius:   req.TempCelsius,
		MinAcceptable: req.MinAcceptable,
		MaxAcceptable: req.MaxAcceptable,
		Location:      req.Location,
	}

	created, err := h.repo.AddTemperatureReading(r.Context(), reading)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to add temperature reading")
		return
	}

	h.asyncRecalculate(batchID)

	WriteJSON(w, http.StatusCreated, created)
}

// AddQualityCheck handles POST /api/producer/batches/{id}/quality-checks.
func (h *ProducerHandler) AddQualityCheck(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	batchID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid batch id: must be a valid UUID")
		return
	}

	var req struct {
		CheckType string  `json:"check_type"`
		Passed    bool    `json:"passed"`
		CheckedAt string  `json:"checked_at"`
		Inspector *string `json:"inspector,omitempty"`
		Notes     *string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CheckType == "" {
		WriteError(w, http.StatusBadRequest, "check_type is required")
		return
	}

	checkedAt, err := time.Parse(time.RFC3339, req.CheckedAt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid checked_at: must be RFC3339")
		return
	}

	check := domain.QualityCheck{
		BatchID:   batchID,
		CheckType: req.CheckType,
		Passed:    req.Passed,
		CheckedAt: checkedAt,
		Inspector: req.Inspector,
		Notes:     req.Notes,
	}

	created, err := h.repo.AddQualityCheck(r.Context(), check)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to add quality check")
		return
	}

	h.asyncRecalculate(batchID)

	WriteJSON(w, http.StatusCreated, created)
}
