package domain

import "time"

// ScanResponse is the single-call payload returned by GET /api/scan/{barcode}.
type ScanResponse struct {
	Product    ScanProduct        `json:"product"`
	Batch      ScanBatch          `json:"batch"`
	TrustScore ScanTrustScore     `json:"trust_score"`
	Journey    []ScanJourneyStep  `json:"journey"`
	Recall     *ScanRecall        `json:"recall"`
	Certs      []ScanCertification `json:"certifications"`
	Sustain    *ScanSustainability `json:"sustainability"`
	Anomalies  []Anomaly           `json:"anomalies,omitempty"`
}

type ScanProduct struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Category string       `json:"category"`
	Barcode  string       `json:"barcode"`
	Producer ScanProducer `json:"producer"`
}

type ScanProducer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Country  string `json:"country"`
}

type ScanBatch struct {
	ID             string  `json:"id"`
	LotNumber      string  `json:"lot_number"`
	ProductionDate string  `json:"production_date"`
	ExpiryDate     *string `json:"expiry_date,omitempty"`
}

type ScanTrustScore struct {
	Overall      float64              `json:"overall"`
	Label        string               `json:"label"`
	Color        string               `json:"color"`
	CalculatedAt *time.Time           `json:"calculated_at,omitempty"`
	SubScores    ScanTrustSubScores   `json:"sub_scores"`
}

type ScanTrustSubScores struct {
	ColdChain          *ScanSubScore `json:"cold_chain,omitempty"`
	QualityChecks      *ScanSubScore `json:"quality_checks,omitempty"`
	TimeToShelf        *ScanSubScore `json:"time_to_shelf,omitempty"`
	ProducerTrackRecord *ScanSubScore `json:"producer_track_record,omitempty"`
	HandlingSteps      *ScanSubScore `json:"handling_steps,omitempty"`
}

type ScanSubScore struct {
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
}

type ScanJourneyStep struct {
	StepOrder  int     `json:"step_order"`
	StepType   string  `json:"step_type"`
	Location   string  `json:"location"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	ArrivedAt  string  `json:"arrived_at"`
	DepartedAt *string `json:"departed_at,omitempty"`
}

type ScanRecall struct {
	Severity     string `json:"severity"`
	Reason       string `json:"reason"`
	Instructions string `json:"instructions"`
	RecalledAt   string `json:"recalled_at"`
	IsActive     bool   `json:"is_active"`
}

type ScanCertification struct {
	CertType    string  `json:"cert_type"`
	IssuingBody string  `json:"issuing_body"`
	ValidUntil  *string `json:"valid_until,omitempty"`
}

type ScanSustainability struct {
	CO2Kg       *float64 `json:"co2_kg,omitempty"`
	WaterLiters *float64 `json:"water_liters,omitempty"`
	TransportKm *float64 `json:"transport_km,omitempty"`
}
