package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/halva2251/trackmyfood-backend/internal/service"
)

func TestTrustScoreService_calcTimeToShelf_WithData(t *testing.T) {
	batchID := uuid.New()
	producerID := uuid.New()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Production 12h ago, delivered 12h ago, optimal=24h
	// ratio = 24/12 = 2.0, capped at 1.0 → score = 100
	now := time.Now().UTC()
	productionDate := now.Add(-12 * time.Hour)
	deliveredAt := now

	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))

	// Cold chain: 5 total, 5 in range
	mock.ExpectQuery("SELECT").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"total", "in_range"}).AddRow(5, 5))

	// Quality: 2 total, 2 passed
	mock.ExpectQuery("SELECT").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"total", "passed"}).AddRow(2, 2))

	// Time to shelf: production 12h ago, optimal 24h, delivered now
	optimalHours := 24
	mock.ExpectQuery("SELECT b.production_date").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"production_date", "optimal_shelf_hours", "delivered_at"}).
			AddRow(productionDate, &optimalHours, &deliveredAt))

	// Producer track record: get producer ID
	mock.ExpectQuery("SELECT pr.id").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))

	// Producer stats
	mock.ExpectQuery("SELECT").
		WithArgs(producerID).
		WillReturnRows(pgxmock.NewRows([]string{"total_batches", "total_recalls", "total_complaints"}).AddRow(3, 0, 0))

	// Handling: 2 steps, optimal = 2
	optSteps := 2
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"actual_steps", "optimal_handling_steps"}).AddRow(2, &optSteps))

	// No recall
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	// Update — overall should be 100
	mock.ExpectExec("UPDATE batches SET").
		WithArgs(batchID, float64(100), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	svc := service.NewTrustScoreService(mock)
	err = svc.Recalculate(context.Background(), batchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestTrustScoreService_PartialSubScores_WeightRedistributed(t *testing.T) {
	batchID := uuid.New()
	producerID := uuid.New()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Only cold chain and quality have data (time-to-shelf nil, producer nil, handling nil)
	// Weights: cold_chain=0.30, quality=0.25 → total=0.55 → redistribute
	// cold chain=100, quality=100 → overall should be 100

	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))

	// Cold chain: 10 total, 10 in range → 100
	mock.ExpectQuery("SELECT").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"total", "in_range"}).AddRow(10, 10))

	// Quality: 4 total, 4 passed → 100
	mock.ExpectQuery("SELECT").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"total", "passed"}).AddRow(4, 4))

	// Time to shelf: no data (nil optimalHours, nil deliveredAt)
	mock.ExpectQuery("SELECT b.production_date").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"production_date", "optimal_shelf_hours", "delivered_at"}).
			AddRow(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), nil, nil))

	// Producer: get ID
	mock.ExpectQuery("SELECT pr.id").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))

	// Producer stats: 0 batches → no data
	mock.ExpectQuery("SELECT").
		WithArgs(producerID).
		WillReturnRows(pgxmock.NewRows([]string{"total_batches", "total_recalls", "total_complaints"}).AddRow(0, 0, 0))

	// Handling: no rows
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(batchID).
		WillReturnError(pgx.ErrNoRows)

	// No recall
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	// Update — overall should be 100 (cold chain 100 + quality 100, weights redistributed)
	mock.ExpectExec("UPDATE batches SET").
		WithArgs(batchID, float64(100), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	svc := service.NewTrustScoreService(mock)
	err = svc.Recalculate(context.Background(), batchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestTrustScoreService_Recalculate(t *testing.T) {
	batchID := uuid.New()
	producerID := uuid.New()

	t.Run("perfect batch — all sub-scores present, no recall", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))

		// Cold chain: 10 total, 10 in range → 100%
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "in_range"}).AddRow(10, 10))

		// Quality: 4 total, 4 passed → 100%
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "passed"}).AddRow(4, 4))

		// Time to shelf: production_date, optimal_shelf_hours, delivered_at
		// Use mock that returns -1 (no data) to simplify — NULL delivered_at
		mock.ExpectQuery("SELECT b.production_date").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"production_date", "optimal_shelf_hours", "delivered_at"}).
				AddRow(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, nil))

		// Producer track record: get producer ID
		mock.ExpectQuery("SELECT pr.id").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))

		// Producer stats: 4 batches, 0 recalls, 0 complaints → 100
		mock.ExpectQuery("SELECT").
			WithArgs(producerID).
			WillReturnRows(pgxmock.NewRows([]string{"total_batches", "total_recalls", "total_complaints"}).AddRow(4, 0, 0))

		// Handling: 3 steps, optimal = 3 → 100%
		optSteps := 3
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"actual_steps", "optimal_handling_steps"}).AddRow(3, &optSteps))

		// Check recall — no active recall
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		// Update batches with calculated scores
		mock.ExpectExec("UPDATE batches SET").
			WithArgs(batchID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectCommit()

		svc := service.NewTrustScoreService(mock)
		err = svc.Recalculate(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("recalled batch — score overridden to 0", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))

		// Cold chain: 10 total, 10 in range
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "in_range"}).AddRow(10, 10))

		// Quality: 3 total, 3 passed
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "passed"}).AddRow(3, 3))

		// Time to shelf: no data
		mock.ExpectQuery("SELECT b.production_date").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"production_date", "optimal_shelf_hours", "delivered_at"}).
				AddRow(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, nil))

		// Producer track record
		mock.ExpectQuery("SELECT pr.id").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))
		mock.ExpectQuery("SELECT").
			WithArgs(producerID).
			WillReturnRows(pgxmock.NewRows([]string{"total_batches", "total_recalls", "total_complaints"}).AddRow(4, 0, 0))

		// Handling: no data
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"actual_steps", "optimal_handling_steps"}).AddRow(0, nil))

		// Check recall — HAS active recall
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		// Update — overall should be 0 due to recall
		mock.ExpectExec("UPDATE batches SET").
			WithArgs(batchID, float64(0), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectCommit()

		svc := service.NewTrustScoreService(mock)
		err = svc.Recalculate(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("no data for any sub-score — overall is 0", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))

		// Cold chain: 0 readings
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "in_range"}).AddRow(0, 0))

		// Quality: 0 checks
		mock.ExpectQuery("SELECT").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"total", "passed"}).AddRow(0, 0))

		// Time to shelf: no data
		mock.ExpectQuery("SELECT b.production_date").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"production_date", "optimal_shelf_hours", "delivered_at"}).
				AddRow(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, nil))

		// Producer: get ID
		mock.ExpectQuery("SELECT pr.id").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))
		// Producer stats: 0 batches
		mock.ExpectQuery("SELECT").
			WithArgs(producerID).
			WillReturnRows(pgxmock.NewRows([]string{"total_batches", "total_recalls", "total_complaints"}).AddRow(0, 0, 0))

		// Handling: no rows (pgx.ErrNoRows → returns -1, nil)
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(batchID).
			WillReturnError(pgx.ErrNoRows)

		// No active recall
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		// Update — all sub-scores are -1, so overall should be 0
		mock.ExpectExec("UPDATE batches SET").
			WithArgs(batchID, float64(0), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectCommit()

		svc := service.NewTrustScoreService(mock)
		err = svc.Recalculate(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}
