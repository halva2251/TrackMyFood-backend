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

// Transactor is an interface for something that can start a transaction.
type Transactor interface {
	DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type TrustScoreService struct {
	db DBTX
}

func NewTrustScoreService(db DBTX) *TrustScoreService {
	return &TrustScoreService{db: db}
}

type subScoreEntry struct {
	score  *float64
	weight float64
}

func (s *TrustScoreService) Recalculate(ctx context.Context, batchID uuid.UUID) error {
	// Use a transaction for the entire recalculation to support advisory locks
	var tx pgx.Tx
	var err error
	var startedTx bool

	if pool, ok := s.db.(Transactor); ok {
		tx, err = pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)
		startedTx = true
	} else if t, ok := s.db.(pgx.Tx); ok {
		tx = t
	} else {
		return fmt.Errorf("recalculate requires a Transactor (e.g. *pgxpool.Pool) or pgx.Tx")
	}

	// 1. Acquire a transaction-level advisory lock for this batch.
	// This ensures only one recalculation happens at a time for a specific batch.
	// We use the first 8 bytes of the UUID as the lock ID.
	lockID := int64(batchID[0])<<56 | int64(batchID[1])<<48 | int64(batchID[2])<<40 | int64(batchID[3])<<32 |
		int64(batchID[4])<<24 | int64(batchID[5])<<16 | int64(batchID[6])<<8 | int64(batchID[7])
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}

	coldChain, err := s.calcColdChainTx(ctx, tx, batchID)
	if err != nil {
		return fmt.Errorf("cold chain: %w", err)
	}

	quality, err := s.calcQualityTx(ctx, tx, batchID)
	if err != nil {
		return fmt.Errorf("quality: %w", err)
	}

	timeToShelf, err := s.calcTimeToShelfTx(ctx, tx, batchID)
	if err != nil {
		return fmt.Errorf("time to shelf: %w", err)
	}

	producer, err := s.calcProducerTrackRecordTx(ctx, tx, batchID)
	if err != nil {
		return fmt.Errorf("producer track record: %w", err)
	}

	handling, err := s.calcHandlingTx(ctx, tx, batchID)
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

	// Filter out entries with nil (no data) and redistribute weights
	var active []subScoreEntry
	for _, e := range entries {
		if e.score != nil {
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
			overall += *e.score * (e.weight / totalWeight)
		}
	}

	overall = math.Round(overall*100) / 100

	// Check for active recall — override to 0
	var hasRecall bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM recalls WHERE batch_id = $1 AND is_active = true)`, batchID).Scan(&hasRecall)
	if err != nil {
		return fmt.Errorf("check recall: %w", err)
	}
	if hasRecall {
		overall = 0
	}

	const update = `
		UPDATE batches SET
			trust_score = $2,
			sub_score_cold_chain = $3,
			sub_score_quality = $4,
			sub_score_time_to_shelf = $5,
			sub_score_producer = $6,
			sub_score_handling = $7,
			score_calculated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, update, batchID, overall, coldChain, quality, timeToShelf, producer, handling)
	if err != nil {
		return fmt.Errorf("update batch: %w", err)
	}

	// Only commit if we started the transaction here
	if startedTx {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
	}

	return nil
}

func ptr(v float64) *float64 { return &v }

func (s *TrustScoreService) calcColdChainTx(ctx context.Context, db DBTX, batchID uuid.UUID) (*float64, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE temp_celsius BETWEEN min_acceptable AND max_acceptable) AS in_range
		FROM temperature_readings
		WHERE batch_id = $1
	`
	var total, inRange int
	if err := db.QueryRow(ctx, q, batchID).Scan(&total, &inRange); err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}
	return ptr(math.Round(float64(inRange)/float64(total)*10000) / 100), nil
}

func (s *TrustScoreService) calcQualityTx(ctx context.Context, db DBTX, batchID uuid.UUID) (*float64, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE passed = true) AS passed
		FROM quality_checks
		WHERE batch_id = $1
	`
	var total, passed int
	if err := db.QueryRow(ctx, q, batchID).Scan(&total, &passed); err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}
	return ptr(math.Round(float64(passed)/float64(total)*10000) / 100), nil
}

func (s *TrustScoreService) calcTimeToShelfTx(ctx context.Context, db DBTX, batchID uuid.UUID) (*float64, error) {
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
	if err := db.QueryRow(ctx, q, batchID).Scan(&productionDate, &optimalHours, &deliveredAt); err != nil {
		return nil, err
	}
	if optimalHours == nil || deliveredAt == nil || *optimalHours == 0 {
		return nil, nil
	}

	actualHours := deliveredAt.Sub(productionDate).Hours()
	if actualHours <= 0 {
		return ptr(100), nil
	}

	ratio := float64(*optimalHours) / actualHours
	if ratio > 1 {
		ratio = 1
	}
	return ptr(math.Round(ratio * 10000) / 100), nil
}

func (s *TrustScoreService) calcProducerTrackRecordTx(ctx context.Context, db DBTX, batchID uuid.UUID) (*float64, error) {
	const q = `
		SELECT pr.id
		FROM batches b
		JOIN products p ON p.id = b.product_id
		JOIN producers pr ON pr.id = p.producer_id
		WHERE b.id = $1
	`
	var producerID uuid.UUID
	if err := db.QueryRow(ctx, q, batchID).Scan(&producerID); err != nil {
		return nil, err
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
	if err := db.QueryRow(ctx, statsQ, producerID).Scan(&totalBatches, &totalRecalls, &totalComplaints); err != nil {
		return nil, err
	}
	if totalBatches == 0 {
		return nil, nil
	}

	// Start at 100, deduct for recalls and complaints
	score := 100.0
	recallPenalty := float64(totalRecalls) / float64(totalBatches) * 50
	complaintPenalty := float64(totalComplaints) / float64(totalBatches) * 30
	score -= recallPenalty + complaintPenalty

	if score < 0 {
		score = 0
	}
	return ptr(math.Round(score*100) / 100), nil
}

func (s *TrustScoreService) calcHandlingTx(ctx context.Context, db DBTX, batchID uuid.UUID) (*float64, error) {
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
	err := db.QueryRow(ctx, q, batchID).Scan(&actualSteps, &optimalSteps)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if optimalSteps == nil {
		return nil, nil
	}
	if *optimalSteps == 0 {
		if actualSteps == 0 {
			return ptr(100), nil
		}
		return ptr(0), nil
	}

	ratio := float64(*optimalSteps) / float64(actualSteps)
	if ratio > 1 {
		ratio = 1
	}
	return ptr(math.Round(ratio * 10000) / 100), nil
}
