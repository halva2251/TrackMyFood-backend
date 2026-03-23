package domain

import (
	"time"

	"github.com/google/uuid"
)

type Recall struct {
	ID           uuid.UUID `json:"id"`
	BatchID      uuid.UUID `json:"batch_id"`
	Severity     string    `json:"severity"`
	Reason       string    `json:"reason"`
	Instructions string    `json:"instructions"`
	RecalledAt   time.Time `json:"recalled_at"`
	IsActive     bool      `json:"is_active"`
}
