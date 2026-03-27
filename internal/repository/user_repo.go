package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type UserRepo struct {
	db DBTX
}

func NewUserRepo(db DBTX) *UserRepo {
	return &UserRepo{db: db}
}

type UserWithHash struct {
	domain.User
	PasswordHash string
}

func (r *UserRepo) Create(ctx context.Context, email, displayName, passwordHash string) (*domain.User, error) {
	const q = `
		INSERT INTO users (email, display_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, display_name, created_at
	`
	var user domain.User
	err := r.db.QueryRow(ctx, q, email, displayName, passwordHash).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*UserWithHash, error) {
	const q = `
		SELECT id, email, display_name, password_hash, created_at
		FROM users
		WHERE email = $1
	`
	var u UserWithHash
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, display_name, created_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, q, id).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

type ScanHistoryEntry struct {
	ID          uuid.UUID `json:"id"`
	ScannedAt   time.Time `json:"scanned_at"`
	ProductName string    `json:"product_name"`
	Category    string    `json:"category"`
	Barcode     string    `json:"barcode"`
	LotNumber   string    `json:"lot_number"`
	TrustScore  *float64  `json:"trust_score"`
	TrustLabel  string    `json:"trust_label"`
	TrustColor  string    `json:"trust_color"`
	BatchID     uuid.UUID `json:"batch_id"`
}

func (r *UserRepo) GetScanHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]ScanHistoryEntry, int, error) {
	const countQ = `SELECT COUNT(*) FROM scan_history WHERE user_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan history: %w", err)
	}

	const q = `
		SELECT sh.id, sh.scanned_at,
			   p.name, p.category, p.barcode,
			   b.lot_number, b.trust_score, b.id
		FROM scan_history sh
		JOIN batches b ON b.id = sh.batch_id
		JOIN products p ON p.id = b.product_id
		WHERE sh.user_id = $1
		ORDER BY sh.scanned_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query scan history: %w", err)
	}
	defer rows.Close()

	entries := make([]ScanHistoryEntry, 0)
	for rows.Next() {
		var e ScanHistoryEntry
		if err := rows.Scan(
			&e.ID, &e.ScannedAt,
			&e.ProductName, &e.Category, &e.Barcode,
			&e.LotNumber, &e.TrustScore, &e.BatchID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan history row: %w", err)
		}
		score := 0.0
		if e.TrustScore != nil {
			score = *e.TrustScore
		}
		e.TrustLabel = domain.TrustScoreLabel(score)
		e.TrustColor = domain.TrustScoreColor(score)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("scan history rows: %w", err)
	}

	return entries, total, nil
}

// DeleteScanHistoryEntry removes a single scan history entry owned by the user.
func (r *UserRepo) DeleteScanHistoryEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	const q = `DELETE FROM scan_history WHERE id = $1 AND user_id = $2`
	tag, err := r.db.Exec(ctx, q, entryID, userID)
	if err != nil {
		return fmt.Errorf("delete scan history entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
