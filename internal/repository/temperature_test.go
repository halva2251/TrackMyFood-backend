package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/halva2251/trackmyfood-backend/internal/repository"
)

func TestTemperatureRepo_GetByBatchID(t *testing.T) {
	batchID := uuid.New()
	now := time.Now().Truncate(time.Second)
	loc := "Cold Storage A"

	t.Run("returns readings ordered by recorded_at", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		rows := mock.NewRows([]string{"id", "batch_id", "recorded_at", "temp_celsius", "min_acceptable", "max_acceptable", "location"}).
			AddRow(uuid.New(), batchID, now, 2.5, 0.0, 4.0, &loc).
			AddRow(uuid.New(), batchID, now.Add(time.Hour), 3.1, 0.0, 4.0, &loc)

		mock.ExpectQuery("SELECT id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location").
			WithArgs(batchID).
			WillReturnRows(rows)

		repo := repository.NewTemperatureRepo(mock)
		readings, err := repo.GetByBatchID(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(readings) != 2 {
			t.Fatalf("got %d readings, want 2", len(readings))
		}
		if readings[0].TempCelsius != 2.5 {
			t.Errorf("first reading temp = %v, want 2.5", readings[0].TempCelsius)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("returns empty slice when no readings", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		rows := mock.NewRows([]string{"id", "batch_id", "recorded_at", "temp_celsius", "min_acceptable", "max_acceptable", "location"})
		mock.ExpectQuery("SELECT id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location").
			WithArgs(batchID).
			WillReturnRows(rows)

		repo := repository.NewTemperatureRepo(mock)
		readings, err := repo.GetByBatchID(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(readings) != 0 {
			t.Fatalf("got %d readings, want 0", len(readings))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("returns error on query failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location").
			WithArgs(batchID).
			WillReturnError(fmt.Errorf("connection refused"))

		repo := repository.NewTemperatureRepo(mock)
		_, err = repo.GetByBatchID(context.Background(), batchID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}
