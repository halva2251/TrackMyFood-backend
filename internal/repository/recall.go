package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type RecallRepo struct {
	db *pgxpool.Pool
}

func NewRecallRepo(db *pgxpool.Pool) *RecallRepo {
	return &RecallRepo{db: db}
}

func (r *RecallRepo) Create(ctx context.Context, recall domain.Recall) (domain.Recall, error) {
	const q = `
		INSERT INTO recalls (batch_id, severity, reason, instructions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, recalled_at, is_active
	`
	err := r.db.QueryRow(ctx, q, recall.BatchID, recall.Severity, recall.Reason, recall.Instructions).
		Scan(&recall.ID, &recall.RecalledAt, &recall.IsActive)
	if err != nil {
		return domain.Recall{}, fmt.Errorf("insert recall: %w", err)
	}
	return recall, nil
}

func (r *RecallRepo) ZeroBatchScore(ctx context.Context, batchID uuid.UUID) error {
	const q = `UPDATE batches SET trust_score = 0, score_calculated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, batchID)
	return err
}

func (r *RecallRepo) GetAffectedUsers(ctx context.Context, batchID uuid.UUID) ([]domain.User, error) {
	const q = `
		SELECT DISTINCT u.id, u.email, u.display_name, u.created_at
		FROM scan_history sh
		JOIN users u ON u.id = sh.user_id
		WHERE sh.batch_id = $1
	`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, fmt.Errorf("query affected users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
