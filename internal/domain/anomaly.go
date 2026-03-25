package domain

// Anomaly represents a statistical deviation for a batch metric vs. its product category average.
type Anomaly struct {
	MetricName     string  `json:"metric_name"`
	BatchValue     float64 `json:"batch_value"`
	CategoryMean   float64 `json:"category_mean"`
	CategoryStdDev float64 `json:"category_stddev"`
	ZScore         float64 `json:"z_score"`
	IsAnomaly      bool    `json:"is_anomaly"`
	Description    string  `json:"description"`
}
