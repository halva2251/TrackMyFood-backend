package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type TemperatureRepo struct {
	db DBTX
}

func NewTemperatureRepo(db DBTX) *TemperatureRepo {
	return &TemperatureRepo{db: db}
}

func (r *TemperatureRepo) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]domain.TemperatureReading, error) {
	const q = `
		SELECT id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location
		FROM temperature_readings
		WHERE batch_id = $1
		ORDER BY recorded_at
	`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, fmt.Errorf("query temperature readings: %w", err)
	}
	defer rows.Close()

	var readings []domain.TemperatureReading
	for rows.Next() {
		var t domain.TemperatureReading
		if err := rows.Scan(&t.ID, &t.BatchID, &t.RecordedAt, &t.TempCelsius, &t.MinAcceptable, &t.MaxAcceptable, &t.Location); err != nil {
			return nil, fmt.Errorf("scan temperature reading: %w", err)
		}
		readings = append(readings, t)
	}
	return readings, rows.Err()
}
