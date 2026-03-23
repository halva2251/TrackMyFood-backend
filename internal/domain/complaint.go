package domain

import (
	"time"

	"github.com/google/uuid"
)

type Complaint struct {
	ID            uuid.UUID `json:"id"`
	BatchID       uuid.UUID `json:"batch_id"`
	UserID        uuid.UUID `json:"user_id"`
	ComplaintType string    `json:"complaint_type"`
	Description   *string   `json:"description,omitempty"`
	PhotoURL      *string   `json:"photo_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
