package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
