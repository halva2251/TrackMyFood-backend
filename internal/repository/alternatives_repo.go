package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

// AlternativesRepo retrieves alternative products in the same category.
type AlternativesRepo struct {
	db DBTX
}

func NewAlternativesRepo(db DBTX) *AlternativesRepo {
	return &AlternativesRepo{db: db}
}

// GetAlternatives returns up to 5 products in the same category as the given product,
// with a higher trust score than minScore, ordered by trust score descending.
// It excludes the product itself.
func (r *AlternativesRepo) GetAlternatives(ctx context.Context, productID uuid.UUID, category string, minScore float64) ([]domain.ScanProduct, error) {
	const q = `
		SELECT p.id, p.name, p.category, p.barcode,
		       pr.id, pr.name, pr.location, pr.country,
		       MAX(b.trust_score) AS best_score
		FROM products p
		JOIN producers pr ON pr.id = p.producer_id
		JOIN batches b ON b.product_id = p.id
		WHERE p.category = $1
		  AND p.id != $2
		  AND b.trust_score > $3
		  AND b.trust_score IS NOT NULL
		GROUP BY p.id, p.name, p.category, p.barcode, pr.id, pr.name, pr.location, pr.country
		ORDER BY best_score DESC
		LIMIT 5
	`
	rows, err := r.db.Query(ctx, q, category, productID, minScore)
	if err != nil {
		return nil, fmt.Errorf("query alternatives: %w", err)
	}
	defer rows.Close()

	results := make([]domain.ScanProduct, 0)
	for rows.Next() {
		var p domain.ScanProduct
		var pr domain.ScanProducer
		var productID, producerID uuid.UUID
		var bestScore float64
		if err := rows.Scan(
			&productID, &p.Name, &p.Category, &p.Barcode,
			&producerID, &pr.Name, &pr.Location, &pr.Country,
			&bestScore,
		); err != nil {
			return nil, fmt.Errorf("scan alternative row: %w", err)
		}
		p.ID = productID.String()
		pr.ID = producerID.String()
		p.Producer = pr
		results = append(results, p)
	}
	return results, rows.Err()
}
