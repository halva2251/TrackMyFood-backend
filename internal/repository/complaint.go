package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ComplaintRepo struct {
	db DBTX
}

func NewComplaintRepo(db DBTX) *ComplaintRepo {
	return &ComplaintRepo{db: db}
}

func (r *ComplaintRepo) Create(ctx context.Context, c domain.Complaint) (domain.Complaint, error) {
	const q = `
		INSERT INTO complaints (batch_id, user_id, complaint_type, description, photo_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, q, c.BatchID, c.UserID, c.ComplaintType, c.Description, c.PhotoURL).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return domain.Complaint{}, fmt.Errorf("insert complaint: %w", err)
	}
	return c, nil
}

func (r *ComplaintRepo) GetProducerIDByBatchID(ctx context.Context, batchID uuid.UUID) (uuid.UUID, error) {
	const q = `
		SELECT pr.id
		FROM batches b
		JOIN products p ON p.id = b.product_id
		JOIN producers pr ON pr.id = p.producer_id
		WHERE b.id = $1
	`
	var producerID uuid.UUID
	err := r.db.QueryRow(ctx, q, batchID).Scan(&producerID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get producer by batch: %w", err)
	}
	return producerID, nil
}
