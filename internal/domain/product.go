package domain

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID                  uuid.UUID `json:"id"`
	ProducerID          uuid.UUID `json:"producer_id"`
	Name                string    `json:"name"`
	Category            string    `json:"category"`
	Barcode             string    `json:"barcode"`
	MinTempCelsius      *float64  `json:"min_temp_celsius,omitempty"`
	MaxTempCelsius      *float64  `json:"max_temp_celsius,omitempty"`
	OptimalShelfHours   *int      `json:"optimal_shelf_hours,omitempty"`
	OptimalHandlingSteps *int     `json:"optimal_handling_steps,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
