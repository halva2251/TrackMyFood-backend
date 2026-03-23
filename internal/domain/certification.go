package domain

import (
	"time"

	"github.com/google/uuid"
)

type Certification struct {
	ID          uuid.UUID  `json:"id"`
	BatchID     uuid.UUID  `json:"batch_id"`
	CertType    string     `json:"cert_type"`
	IssuingBody string     `json:"issuing_body"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
}
