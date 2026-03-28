package domain

type LeaderboardEntry struct {
	Rank           int      `json:"rank"`
	ProductName    string   `json:"product_name"`
	ProducerName   string   `json:"producer_name"`
	Country        string   `json:"country"`
	Category       string   `json:"category"`
	LotNumber      string   `json:"lot_number,omitempty"`
	Barcode        string   `json:"barcode"`
	TrustScore     float64  `json:"trust_score"`
	ColdChainScore *float64 `json:"cold_chain_score,omitempty"`
	QualityScore   *float64 `json:"quality_score,omitempty"`
}

type LeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
	Count       int                `json:"count"`
}
