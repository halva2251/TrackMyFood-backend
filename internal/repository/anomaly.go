package repository

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type AnomalyRepo struct {
	db DBTX
}

func NewAnomalyRepo(db DBTX) *AnomalyRepo {
	return &AnomalyRepo{db: db}
}

// DetectAnomalies computes z-scores for this batch's metrics vs. its product category averages.
// A metric is flagged as anomalous if |z-score| > 2 and stddev > 0.
func (r *AnomalyRepo) DetectAnomalies(ctx context.Context, batchID uuid.UUID) ([]domain.Anomaly, error) {
	// Single query: compute batch-level metrics and category-level stats in one shot.
	const q = `
		WITH batch_info AS (
			SELECT b.id AS batch_id, p.id AS product_id, p.category
			FROM batches b
			JOIN products p ON p.id = b.product_id
			WHERE b.id = $1
		),
		-- Cold chain compliance per batch (% of readings in range)
		cold_chain AS (
			SELECT
				tr.batch_id,
				CASE WHEN COUNT(*) = 0 THEN NULL
					ELSE COUNT(*) FILTER (WHERE tr.temp_celsius BETWEEN tr.min_acceptable AND tr.max_acceptable)::float / COUNT(*)::float * 100
				END AS compliance
			FROM temperature_readings tr
			JOIN batches b ON b.id = tr.batch_id
			JOIN products p ON p.id = b.product_id
			WHERE p.category = (SELECT category FROM batch_info)
			GROUP BY tr.batch_id
		),
		-- Quality pass rate per batch
		quality AS (
			SELECT
				qc.batch_id,
				CASE WHEN COUNT(*) = 0 THEN NULL
					ELSE COUNT(*) FILTER (WHERE qc.passed = true)::float / COUNT(*)::float * 100
				END AS pass_rate
			FROM quality_checks qc
			JOIN batches b ON b.id = qc.batch_id
			JOIN products p ON p.id = b.product_id
			WHERE p.category = (SELECT category FROM batch_info)
			GROUP BY qc.batch_id
		),
		-- Handling steps per batch
		handling AS (
			SELECT
				js.batch_id,
				COUNT(*)::float AS steps
			FROM journey_steps js
			JOIN batches b ON b.id = js.batch_id
			JOIN products p ON p.id = b.product_id
			WHERE p.category = (SELECT category FROM batch_info)
			GROUP BY js.batch_id
		)
		SELECT
			-- Cold chain stats
			(SELECT compliance FROM cold_chain WHERE batch_id = $1) AS batch_cold_chain,
			(SELECT AVG(compliance) FROM cold_chain WHERE compliance IS NOT NULL) AS avg_cold_chain,
			(SELECT STDDEV(compliance) FROM cold_chain WHERE compliance IS NOT NULL) AS std_cold_chain,
			-- Quality stats
			(SELECT pass_rate FROM quality WHERE batch_id = $1) AS batch_quality,
			(SELECT AVG(pass_rate) FROM quality WHERE pass_rate IS NOT NULL) AS avg_quality,
			(SELECT STDDEV(pass_rate) FROM quality WHERE pass_rate IS NOT NULL) AS std_quality,
			-- Handling steps stats
			(SELECT steps FROM handling WHERE batch_id = $1) AS batch_handling,
			(SELECT AVG(steps) FROM handling) AS avg_handling,
			(SELECT STDDEV(steps) FROM handling) AS std_handling
	`

	var (
		batchColdChain, avgColdChain, stdColdChain *float64
		batchQuality, avgQuality, stdQuality       *float64
		batchHandling, avgHandling, stdHandling    *float64
	)

	err := r.db.QueryRow(ctx, q, batchID).Scan(
		&batchColdChain, &avgColdChain, &stdColdChain,
		&batchQuality, &avgQuality, &stdQuality,
		&batchHandling, &avgHandling, &stdHandling,
	)
	if err != nil {
		return nil, fmt.Errorf("query anomaly stats: %w", err)
	}

	var anomalies []domain.Anomaly

	if a := checkAnomaly("cold_chain_compliance", batchColdChain, avgColdChain, stdColdChain,
		"Cold chain compliance deviates significantly from category average"); a != nil {
		anomalies = append(anomalies, *a)
	}
	if a := checkAnomaly("quality_pass_rate", batchQuality, avgQuality, stdQuality,
		"Quality check pass rate deviates significantly from category average"); a != nil {
		anomalies = append(anomalies, *a)
	}
	if a := checkAnomaly("handling_steps", batchHandling, avgHandling, stdHandling,
		"Number of handling steps deviates significantly from category average"); a != nil {
		anomalies = append(anomalies, *a)
	}

	return anomalies, nil
}

func checkAnomaly(name string, batchVal, mean, stddev *float64, desc string) *domain.Anomaly {
	if batchVal == nil || mean == nil || stddev == nil || *stddev == 0 {
		return nil
	}

	zScore := (*batchVal - *mean) / *stddev
	zScore = math.Round(zScore*100) / 100
	isAnomaly := math.Abs(zScore) > 2.0

	return &domain.Anomaly{
		MetricName:     name,
		BatchValue:     math.Round(*batchVal*100) / 100,
		CategoryMean:   math.Round(*mean*100) / 100,
		CategoryStdDev: math.Round(*stddev*100) / 100,
		ZScore:         zScore,
		IsAnomaly:      isAnomaly,
		Description:    desc,
	}
}
