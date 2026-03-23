package domain

import (
	"time"

	"github.com/google/uuid"
)

type QualityCheck struct {
	ID        uuid.UUID `json:"id"`
	BatchID   uuid.UUID `json:"batch_id"`
	CheckType string    `json:"check_type"`
	Passed    bool      `json:"passed"`
	CheckedAt time.Time `json:"checked_at"`
	Inspector *string   `json:"inspector,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
}
