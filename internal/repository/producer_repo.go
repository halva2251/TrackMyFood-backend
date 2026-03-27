package repository

import (
	"context"
	"fmt"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

// ProducerRepo handles writes for the producer/IoT supply chain data ingestion.
type ProducerRepo struct {
	db DBTX
}

func NewProducerRepo(db DBTX) *ProducerRepo {
	return &ProducerRepo{db: db}
}

// CreateBatch inserts a new batch and returns it with DB-generated id and created_at.
func (r *ProducerRepo) CreateBatch(ctx context.Context, b domain.Batch) (domain.Batch, error) {
	const q = `
		INSERT INTO batches (product_id, lot_number, production_date, expiry_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, q, b.ProductID, b.LotNumber, b.ProductionDate, b.ExpiryDate).
		Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return domain.Batch{}, fmt.Errorf("insert batch: %w", err)
	}
	return b, nil
}

// AddJourneyStep inserts one journey step for a batch and returns it with generated id.
func (r *ProducerRepo) AddJourneyStep(ctx context.Context, s domain.JourneyStep) (domain.JourneyStep, error) {
	const q = `
		INSERT INTO journey_steps (batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, q,
		s.BatchID, s.StepOrder, s.StepType, s.Location,
		s.Latitude, s.Longitude, s.ArrivedAt, s.DepartedAt, s.Notes,
	).Scan(&s.ID)
	if err != nil {
		return domain.JourneyStep{}, fmt.Errorf("insert journey step: %w", err)
	}
	return s, nil
}

// AddTemperatureReading inserts one temperature reading and returns it with generated id.
func (r *ProducerRepo) AddTemperatureReading(ctx context.Context, tr domain.TemperatureReading) (domain.TemperatureReading, error) {
	const q = `
		INSERT INTO temperature_readings (batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, q,
		tr.BatchID, tr.RecordedAt, tr.TempCelsius, tr.MinAcceptable, tr.MaxAcceptable, tr.Location,
	).Scan(&tr.ID)
	if err != nil {
		return domain.TemperatureReading{}, fmt.Errorf("insert temperature reading: %w", err)
	}
	return tr, nil
}

// AddQualityCheck inserts one quality check and returns it with generated id.
func (r *ProducerRepo) AddQualityCheck(ctx context.Context, qc domain.QualityCheck) (domain.QualityCheck, error) {
	const q = `
		INSERT INTO quality_checks (batch_id, check_type, passed, checked_at, inspector, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, q,
		qc.BatchID, qc.CheckType, qc.Passed, qc.CheckedAt, qc.Inspector, qc.Notes,
	).Scan(&qc.ID)
	if err != nil {
		return domain.QualityCheck{}, fmt.Errorf("insert quality check: %w", err)
	}
	return qc, nil
}
