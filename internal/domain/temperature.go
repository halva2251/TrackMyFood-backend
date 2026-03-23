package domain

import (
	"time"

	"github.com/google/uuid"
)

type TemperatureReading struct {
	ID            uuid.UUID `json:"id"`
	BatchID       uuid.UUID `json:"batch_id"`
	RecordedAt    time.Time `json:"recorded_at"`
	TempCelsius   float64   `json:"temp_celsius"`
	MinAcceptable float64   `json:"min_acceptable"`
	MaxAcceptable float64   `json:"max_acceptable"`
	Location      *string   `json:"location,omitempty"`
}
