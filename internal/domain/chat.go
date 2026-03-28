package domain

type ChatRequest struct {
	Question string `json:"question"`
	Lot      string `json:"lot"` // Optional lot number for specific batch context
}

type ChatResponse struct {
	Answer string `json:"answer"`
}
