package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the common interface satisfied by *pgxpool.Pool, pgx.Tx, and mocks.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type TrustScoreService struct {
	db DBTX
}

func NewTrustScoreService(db DBTX) *TrustScoreService {
	return &TrustScoreService{db: db}
}

type subScoreEntry struct {
	score  float64
	weight float64
}

func (s *TrustScoreService) Recalculate(ctx context.Context, batchID uuid.UUID) error {
	coldChain, err := s.calcColdChain(ctx, batchID)
	if err != nil {
		return fmt.Errorf("cold chain: %w", err)
	}

	quality, err := s.calcQuality(ctx, batchID)
	if err != nil {
		return fmt.Errorf("quality: %w", err)
	}

	timeToShelf, err := s.calcTimeToShelf(ctx, batchID)
	if err != nil {
		return fmt.Errorf("time to shelf: %w", err)
	}

	producer, err := s.calcProducerTrackRecord(ctx, batchID)
	if err != nil {
		return fmt.Errorf("producer track record: %w", err)
	}

	handling, err := s.calcHandling(ctx, batchID)
	if err != nil {
		return fmt.Errorf("handling: %w", err)
	}

	entries := []subScoreEntry{
		{coldChain, 0.30},
		{quality, 0.25},
		{timeToShelf, 0.20},
		{producer, 0.15},
		{handling, 0.10},
	}

	// Filter out entries with -1 (no data) and redistribute weights
	var active []subScoreEntry
	for _, e := range entries {
		if e.score >= 0 {
			active = append(active, e)
		}
	}

	overall := 0.0
	if len(active) > 0 {
		totalWeight := 0.0
		for _, e := range active {
			totalWeight += e.weight
		}
		for _, e := range active {
			overall += e.score * (e.weight / totalWeight)
		}
	}

	overall = math.Round(overall*100) / 100

	// Check for active recall — override to 0
	var hasRecall bool
	err = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM recalls WHERE batch_id = $1 AND is_active = true)`, batchID).Scan(&hasRecall)
	if err != nil {
		return fmt.Errorf("check recall: %w", err)
	}
	if hasRecall {
		overall = 0
	}

	const update = `
		UPDATE batches SET
			trust_score = $2,
			sub_score_cold_chain = CASE WHEN $3::decimal >= 0 THEN $3 ELSE NULL END,
			sub_score_quality = CASE WHEN $4::decimal >= 0 THEN $4 ELSE NULL END,
			sub_score_time_to_shelf = CASE WHEN $5::decimal >= 0 THEN $5 ELSE NULL END,
			sub_score_producer = CASE WHEN $6::decimal >= 0 THEN $6 ELSE NULL END,
			sub_score_handling = CASE WHEN $7::decimal >= 0 THEN $7 ELSE NULL END,
			score_calculated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.Exec(ctx, update, batchID, overall, coldChain, quality, timeToShelf, producer, handling)
	return err
}

// calcColdChain: percentage of readings within acceptable range. Returns -1 if no data.
func (s *TrustScoreService) calcColdChain(ctx context.Context, batchID uuid.UUID) (float64, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE temp_celsius BETWEEN min_acceptable AND max_acceptable) AS in_range
		FROM temperature_readings
		WHERE batch_id = $1
	`
	var total, inRange int
	if err := s.db.QueryRow(ctx, q, batchID).Scan(&total, &inRange); err != nil {
		return 0, err
	}
	if total == 0 {
		return -1, nil
	}
	return math.Round(float64(inRange)/float64(total)*10000) / 100, nil
}

// calcQuality: ratio of passed checks to total checks. Returns -1 if no data.
func (s *TrustScoreService) calcQuality(ctx context.Context, batchID uuid.UUID) (float64, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE passed = true) AS passed
		FROM quality_checks
		WHERE batch_id = $1
	`
	var total, passed int
	if err := s.db.QueryRow(ctx, q, batchID).Scan(&total, &passed); err != nil {
		return 0, err
	}
	if total == 0 {
		return -1, nil
	}
	return math.Round(float64(passed)/float64(total)*10000) / 100, nil
}

// calcTimeToShelf: ratio of optimal hours to actual hours. Returns -1 if no data.
func (s *TrustScoreService) calcTimeToShelf(ctx context.Context, batchID uuid.UUID) (float64, error) {
	const q = `
		SELECT b.production_date, p.optimal_shelf_hours,
			(SELECT MAX(arrived_at) FROM journey_steps WHERE batch_id = b.id AND step_type = 'delivered') AS delivered_at
		FROM batches b
		JOIN products p ON p.id = b.product_id
		WHERE b.id = $1
	`
	var productionDate time.Time
	var optimalHours *int
	var deliveredAt *time.Time
	if err := s.db.QueryRow(ctx, q, batchID).Scan(&productionDate, &optimalHours, &deliveredAt); err != nil {
		return 0, err
	}
	if optimalHours == nil || deliveredAt == nil || *optimalHours == 0 {
		return -1, nil
	}

	actualHours := deliveredAt.Sub(productionDate).Hours()
	if actualHours <= 0 {
		return 100, nil
	}

	ratio := float64(*optimalHours) / actualHours
	if ratio > 1 {
		ratio = 1
	}
	return math.Round(ratio * 10000) / 100, nil
}

// calcProducerTrackRecord: based on complaint rate and recall history. Returns -1 if no data.
func (s *TrustScoreService) calcProducerTrackRecord(ctx context.Context, batchID uuid.UUID) (float64, error) {
	const q = `
		SELECT pr.id
		FROM batches b
		JOIN products p ON p.id = b.product_id
		JOIN producers pr ON pr.id = p.producer_id
		WHERE b.id = $1
	`
	var producerID uuid.UUID
	if err := s.db.QueryRow(ctx, q, batchID).Scan(&producerID); err != nil {
		return 0, err
	}

	const statsQ = `
		SELECT
			COUNT(DISTINCT b2.id) AS total_batches,
			COUNT(DISTINCT r.id) AS total_recalls,
			COUNT(DISTINCT c.id) AS total_complaints
		FROM products p2
		JOIN batches b2 ON b2.product_id = p2.id
		LEFT JOIN recalls r ON r.batch_id = b2.id AND r.is_active = true
		LEFT JOIN complaints c ON c.batch_id = b2.id
		WHERE p2.producer_id = $1
	`
	var totalBatches, totalRecalls, totalComplaints int
	if err := s.db.QueryRow(ctx, statsQ, producerID).Scan(&totalBatches, &totalRecalls, &totalComplaints); err != nil {
		return 0, err
	}
	if totalBatches == 0 {
		return -1, nil
	}

	// Start at 100, deduct for recalls and complaints
	score := 100.0
	recallPenalty := float64(totalRecalls) / float64(totalBatches) * 50
	complaintPenalty := float64(totalComplaints) / float64(totalBatches) * 30
	score -= recallPenalty + complaintPenalty

	if score < 0 {
		score = 0
	}
	return math.Round(score*100) / 100, nil
}

// calcHandling: fewer handling steps = better. Returns -1 if no data.
func (s *TrustScoreService) calcHandling(ctx context.Context, batchID uuid.UUID) (float64, error) {
	const q = `
		SELECT COUNT(*) AS actual_steps, p.optimal_handling_steps
		FROM journey_steps js
		JOIN batches b ON b.id = js.batch_id
		JOIN products p ON p.id = b.product_id
		WHERE js.batch_id = $1
		GROUP BY p.optimal_handling_steps
	`
	var actualSteps int
	var optimalSteps *int
	err := s.db.QueryRow(ctx, q, batchID).Scan(&actualSteps, &optimalSteps)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, nil
		}
		return 0, err
	}
	if optimalSteps == nil || *optimalSteps == 0 {
		return -1, nil
	}

	ratio := float64(*optimalSteps) / float64(actualSteps)
	if ratio > 1 {
		ratio = 1
	}
	return math.Round(ratio * 10000) / 100, nil
}
