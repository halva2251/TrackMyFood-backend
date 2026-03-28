package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type LeaderboardRepo struct {
	db *pgxpool.Pool
}

func NewLeaderboardRepo(db *pgxpool.Pool) *LeaderboardRepo {
	return &LeaderboardRepo{db: db}
}

// GetTopBatches returns the highest-scoring batches ranked by trust score,
// excluding batches with active recalls.
func (r *LeaderboardRepo) GetTopBatches(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	const query = `
		SELECT
			RANK() OVER (ORDER BY b.trust_score DESC) AS rank,
			p.name          AS product_name,
			pr.name         AS producer_name,
			pr.country,
			p.category,
			b.lot_number,
			p.barcode,
			b.trust_score,
			b.sub_score_cold_chain,
			b.sub_score_quality
		FROM batches b
		JOIN products  p  ON p.id  = b.product_id
		JOIN producers pr ON pr.id = p.producer_id
		LEFT JOIN recalls rc ON rc.batch_id = b.id AND rc.is_active = TRUE
		WHERE b.trust_score IS NOT NULL
		  AND rc.id IS NULL
		ORDER BY b.trust_score DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	for rows.Next() {
		var e domain.LeaderboardEntry
		if err := rows.Scan(
			&e.Rank,
			&e.ProductName,
			&e.ProducerName,
			&e.Country,
			&e.Category,
			&e.LotNumber,
			&e.Barcode,
			&e.TrustScore,
			&e.ColdChainScore,
			&e.QualityScore,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []domain.LeaderboardEntry{}
	}

	return entries, rows.Err()
}
