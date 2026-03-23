package domain

import (
	"time"

	"github.com/google/uuid"
)

type Batch struct {
	ID                   uuid.UUID  `json:"id"`
	ProductID            uuid.UUID  `json:"product_id"`
	LotNumber            string     `json:"lot_number"`
	ProductionDate       time.Time  `json:"production_date"`
	ExpiryDate           *time.Time `json:"expiry_date,omitempty"`
	TrustScore           *float64   `json:"trust_score,omitempty"`
	SubScoreColdChain    *float64   `json:"sub_score_cold_chain,omitempty"`
	SubScoreQuality      *float64   `json:"sub_score_quality,omitempty"`
	SubScoreTimeToShelf  *float64   `json:"sub_score_time_to_shelf,omitempty"`
	SubScoreProducer     *float64   `json:"sub_score_producer,omitempty"`
	SubScoreHandling     *float64   `json:"sub_score_handling,omitempty"`
	ScoreCalculatedAt    *time.Time `json:"score_calculated_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// TrustScoreLabel returns a human-readable label for the trust score.
func TrustScoreLabel(score float64) string {
	switch {
	case score >= 80:
		return "Excellent"
	case score >= 60:
		return "Good"
	case score >= 40:
		return "Fair"
	case score >= 20:
		return "Poor"
	default:
		return "Critical"
	}
}

// TrustScoreColor returns a color for the trust score.
func TrustScoreColor(score float64) string {
	switch {
	case score >= 60:
		return "green"
	case score >= 40:
		return "orange"
	default:
		return "red"
	}
}
