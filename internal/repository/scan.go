package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ScanRepo struct {
	db DBTX
}

func NewScanRepo(db DBTX) *ScanRepo {
	return &ScanRepo{db: db}
}

type scanRow struct {
	// product
	ProductID       uuid.UUID
	ProductName     string
	ProductCategory string
	ProductBarcode  string
	// producer
	ProducerID       uuid.UUID
	ProducerName     string
	ProducerLocation string
	ProducerCountry  string
	// batch
	BatchID              uuid.UUID
	LotNumber            string
	ProductionDate       time.Time
	ExpiryDate           *time.Time
	TrustScore           *float64
	SubScoreColdChain    *float64
	SubScoreQuality      *float64
	SubScoreTimeToShelf  *float64
	SubScoreProducer     *float64
	SubScoreHandling     *float64
	ScoreCalculatedAt    *time.Time
}

func (r *ScanRepo) LookupByBarcode(ctx context.Context, barcode string) (*domain.ScanResponse, error) {
	const q = `
		SELECT
			p.id, p.name, p.category, p.barcode,
			pr.id, pr.name, pr.location, pr.country,
			b.id, b.lot_number, b.production_date, b.expiry_date,
			b.trust_score, b.sub_score_cold_chain, b.sub_score_quality,
			b.sub_score_time_to_shelf, b.sub_score_producer, b.sub_score_handling,
			b.score_calculated_at
		FROM products p
		JOIN producers pr ON pr.id = p.producer_id
		JOIN batches b ON b.product_id = p.id
		WHERE p.barcode = $1
		ORDER BY b.production_date DESC
		LIMIT 1
	`

	var row scanRow
	err := r.db.QueryRow(ctx, q, barcode).Scan(
		&row.ProductID, &row.ProductName, &row.ProductCategory, &row.ProductBarcode,
		&row.ProducerID, &row.ProducerName, &row.ProducerLocation, &row.ProducerCountry,
		&row.BatchID, &row.LotNumber, &row.ProductionDate, &row.ExpiryDate,
		&row.TrustScore, &row.SubScoreColdChain, &row.SubScoreQuality,
		&row.SubScoreTimeToShelf, &row.SubScoreProducer, &row.SubScoreHandling,
		&row.ScoreCalculatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query scan by barcode: %w", err)
	}

	score := 0.0
	if row.TrustScore != nil {
		score = *row.TrustScore
	}

	resp := &domain.ScanResponse{
		Product: domain.ScanProduct{
			ID:       row.ProductID.String(),
			Name:     row.ProductName,
			Category: row.ProductCategory,
			Barcode:  row.ProductBarcode,
			Producer: domain.ScanProducer{
				ID:       row.ProducerID.String(),
				Name:     row.ProducerName,
				Location: row.ProducerLocation,
				Country:  row.ProducerCountry,
			},
		},
		Batch: domain.ScanBatch{
			ID:             row.BatchID.String(),
			LotNumber:      row.LotNumber,
			ProductionDate: row.ProductionDate.Format(time.RFC3339),
			ExpiryDate:     formatTimePtr(row.ExpiryDate),
		},
		TrustScore: domain.ScanTrustScore{
			Overall:      score,
			Label:        domain.TrustScoreLabel(score),
			Color:        domain.TrustScoreColor(score),
			CalculatedAt: row.ScoreCalculatedAt,
			SubScores:    buildSubScores(row),
		},
	}

	batchID := row.BatchID

	journey, err := r.getJourney(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("get journey: %w", err)
	}
	resp.Journey = journey

	recall, err := r.getRecall(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("get recall: %w", err)
	}
	resp.Recall = recall

	certs, err := r.getCertifications(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("get certifications: %w", err)
	}
	resp.Certs = certs

	sustain, err := r.getSustainability(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("get sustainability: %w", err)
	}
	resp.Sustain = sustain

	return resp, nil
}

func (r *ScanRepo) RecordScan(ctx context.Context, userID, batchID uuid.UUID) error {
	const q = `INSERT INTO scan_history (user_id, batch_id) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, q, userID, batchID)
	return err
}

func (r *ScanRepo) getJourney(ctx context.Context, batchID uuid.UUID) ([]domain.ScanJourneyStep, error) {
	const q = `
		SELECT step_order, step_type, location, latitude, longitude, arrived_at, departed_at
		FROM journey_steps
		WHERE batch_id = $1
		ORDER BY step_order
	`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []domain.ScanJourneyStep
	for rows.Next() {
		var s domain.ScanJourneyStep
		var arrivedAt time.Time
		var departedAt *time.Time
		if err := rows.Scan(&s.StepOrder, &s.StepType, &s.Location, &s.Latitude, &s.Longitude, &arrivedAt, &departedAt); err != nil {
			return nil, err
		}
		s.ArrivedAt = arrivedAt.Format(time.RFC3339)
		s.DepartedAt = formatTimePtr(departedAt)
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (r *ScanRepo) getRecall(ctx context.Context, batchID uuid.UUID) (*domain.ScanRecall, error) {
	const q = `
		SELECT severity, reason, instructions, recalled_at, is_active
		FROM recalls
		WHERE batch_id = $1 AND is_active = true
	`
	var recall domain.ScanRecall
	var recalledAt time.Time
	err := r.db.QueryRow(ctx, q, batchID).Scan(
		&recall.Severity, &recall.Reason, &recall.Instructions, &recalledAt, &recall.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	recall.RecalledAt = recalledAt.Format(time.RFC3339)
	return &recall, nil
}

func (r *ScanRepo) getCertifications(ctx context.Context, batchID uuid.UUID) ([]domain.ScanCertification, error) {
	const q = `
		SELECT cert_type, issuing_body, valid_until
		FROM certifications
		WHERE batch_id = $1
	`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []domain.ScanCertification
	for rows.Next() {
		var c domain.ScanCertification
		var validUntil *time.Time
		if err := rows.Scan(&c.CertType, &c.IssuingBody, &validUntil); err != nil {
			return nil, err
		}
		if validUntil != nil {
			s := validUntil.Format("2006-01-02")
			c.ValidUntil = &s
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

func (r *ScanRepo) getSustainability(ctx context.Context, batchID uuid.UUID) (*domain.ScanSustainability, error) {
	const q = `
		SELECT co2_kg, water_liters, transport_km
		FROM sustainability
		WHERE batch_id = $1
	`
	var s domain.ScanSustainability
	err := r.db.QueryRow(ctx, q, batchID).Scan(&s.CO2Kg, &s.WaterLiters, &s.TransportKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func buildSubScores(row scanRow) domain.ScanTrustSubScores {
	var ss domain.ScanTrustSubScores
	if row.SubScoreColdChain != nil {
		ss.ColdChain = &domain.ScanSubScore{Score: *row.SubScoreColdChain, Weight: 0.30}
	}
	if row.SubScoreQuality != nil {
		ss.QualityChecks = &domain.ScanSubScore{Score: *row.SubScoreQuality, Weight: 0.25}
	}
	if row.SubScoreTimeToShelf != nil {
		ss.TimeToShelf = &domain.ScanSubScore{Score: *row.SubScoreTimeToShelf, Weight: 0.20}
	}
	if row.SubScoreProducer != nil {
		ss.ProducerTrackRecord = &domain.ScanSubScore{Score: *row.SubScoreProducer, Weight: 0.15}
	}
	if row.SubScoreHandling != nil {
		ss.HandlingSteps = &domain.ScanSubScore{Score: *row.SubScoreHandling, Weight: 0.10}
	}
	return ss
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
