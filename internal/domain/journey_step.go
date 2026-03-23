package domain

import (
	"time"

	"github.com/google/uuid"
)

type JourneyStep struct {
	ID         uuid.UUID  `json:"id"`
	BatchID    uuid.UUID  `json:"batch_id"`
	StepOrder  int        `json:"step_order"`
	StepType   string     `json:"step_type"`
	Location   string     `json:"location"`
	Latitude   *float64   `json:"latitude,omitempty"`
	Longitude  *float64   `json:"longitude,omitempty"`
	ArrivedAt  time.Time  `json:"arrived_at"`
	DepartedAt *time.Time `json:"departed_at,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
}
