package domain

import "github.com/google/uuid"

type Sustainability struct {
	ID          uuid.UUID `json:"id"`
	BatchID     uuid.UUID `json:"batch_id"`
	CO2Kg       *float64  `json:"co2_kg,omitempty"`
	WaterLiters *float64  `json:"water_liters,omitempty"`
	TransportKm *float64  `json:"transport_km,omitempty"`
}
