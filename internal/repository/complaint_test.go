package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/repository"
)

func TestComplaintRepo_Create(t *testing.T) {
	batchID := uuid.New()
	userID := uuid.New()
	complaintID := uuid.New()
	now := time.Now().Truncate(time.Second)

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		desc := "smells bad"
		complaint := domain.Complaint{
			BatchID:       batchID,
			UserID:        userID,
			ComplaintType: "taste_smell",
			Description:   &desc,
		}

		mock.ExpectQuery("INSERT INTO complaints").
			WithArgs(batchID, userID, "taste_smell", &desc, (*string)(nil)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(complaintID, now))

		repo := repository.NewComplaintRepo(mock)
		created, err := repo.Create(context.Background(), complaint)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID != complaintID {
			t.Errorf("id = %v, want %v", created.ID, complaintID)
		}
		if !created.CreatedAt.Equal(now) {
			t.Errorf("created_at = %v, want %v", created.CreatedAt, now)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		complaint := domain.Complaint{
			BatchID:       batchID,
			UserID:        userID,
			ComplaintType: "other",
		}

		mock.ExpectQuery("INSERT INTO complaints").
			WithArgs(batchID, userID, "other", (*string)(nil), (*string)(nil)).
			WillReturnError(fmt.Errorf("unique constraint violation"))

		repo := repository.NewComplaintRepo(mock)
		_, err = repo.Create(context.Background(), complaint)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func TestComplaintRepo_GetProducerIDByBatchID(t *testing.T) {
	batchID := uuid.New()
	producerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery("SELECT pr.id").
			WithArgs(batchID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(producerID))

		repo := repository.NewComplaintRepo(mock)
		got, err := repo.GetProducerIDByBatchID(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != producerID {
			t.Errorf("producer id = %v, want %v", got, producerID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}
