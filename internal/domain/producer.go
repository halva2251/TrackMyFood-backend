package domain

import (
	"time"

	"github.com/google/uuid"
)

type Producer struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at"`
}
