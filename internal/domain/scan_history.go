package domain

import (
	"time"

	"github.com/google/uuid"
)

type ScanHistory struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	BatchID   uuid.UUID `json:"batch_id"`
	ScannedAt time.Time `json:"scanned_at"`
}
