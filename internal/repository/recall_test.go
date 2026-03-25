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

func TestRecallRepo_Create(t *testing.T) {
	batchID := uuid.New()
	recallID := uuid.New()
	now := time.Now().Truncate(time.Second)

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		recall := domain.Recall{
			BatchID:      batchID,
			Severity:     "critical",
			Reason:       "Listeria contamination",
			Instructions: "Do not consume. Return to store.",
		}

		mock.ExpectQuery("INSERT INTO recalls").
			WithArgs(batchID, "critical", "Listeria contamination", "Do not consume. Return to store.").
			WillReturnRows(pgxmock.NewRows([]string{"id", "recalled_at", "is_active"}).
				AddRow(recallID, now, true))

		repo := repository.NewRecallRepo(mock)
		created, err := repo.Create(context.Background(), recall)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID != recallID {
			t.Errorf("id = %v, want %v", created.ID, recallID)
		}
		if !created.IsActive {
			t.Error("expected is_active = true")
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

		mock.ExpectQuery("INSERT INTO recalls").
			WithArgs(batchID, "high", "reason", "instructions").
			WillReturnError(fmt.Errorf("db error"))

		repo := repository.NewRecallRepo(mock)
		_, err = repo.Create(context.Background(), domain.Recall{
			BatchID: batchID, Severity: "high", Reason: "reason", Instructions: "instructions",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRecallRepo_ZeroBatchScore(t *testing.T) {
	batchID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec("UPDATE batches SET trust_score = 0").
			WithArgs(batchID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := repository.NewRecallRepo(mock)
		err = repo.ZeroBatchScore(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func TestRecallRepo_GetAffectedUsers(t *testing.T) {
	batchID := uuid.New()

	t.Run("returns affected users", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		userID := uuid.New()
		now := time.Now().Truncate(time.Second)
		displayName := "Test User"

		rows := mock.NewRows([]string{"id", "email", "display_name", "created_at"}).
			AddRow(userID, "test@example.com", &displayName, now)

		mock.ExpectQuery("SELECT DISTINCT u.id, u.email, u.display_name, u.created_at").
			WithArgs(batchID).
			WillReturnRows(rows)

		repo := repository.NewRecallRepo(mock)
		users, err := repo.GetAffectedUsers(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 1 {
			t.Fatalf("got %d users, want 1", len(users))
		}
		if users[0].Email != "test@example.com" {
			t.Errorf("email = %q, want %q", users[0].Email, "test@example.com")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("returns empty when no users scanned", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		rows := mock.NewRows([]string{"id", "email", "display_name", "created_at"})
		mock.ExpectQuery("SELECT DISTINCT u.id, u.email, u.display_name, u.created_at").
			WithArgs(batchID).
			WillReturnRows(rows)

		repo := repository.NewRecallRepo(mock)
		users, err := repo.GetAffectedUsers(context.Background(), batchID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 0 {
			t.Fatalf("got %d users, want 0", len(users))
		}
	})
}
