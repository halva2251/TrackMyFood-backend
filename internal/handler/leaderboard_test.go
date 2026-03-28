package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type mockLeaderboardRepo struct {
	entries []domain.LeaderboardEntry
	err     error
}

func (m *mockLeaderboardRepo) GetTopBatches(_ context.Context, _ int) ([]domain.LeaderboardEntry, error) {
	return m.entries, m.err
}

func float64Ptr(v float64) *float64 { return &v }

func TestLeaderboardHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		entries    []domain.LeaderboardEntry
		wantStatus int
		wantCount  int
	}{
		{
			name:  "returns ranked entries",
			query: "?limit=10",
			entries: []domain.LeaderboardEntry{
				{Rank: 1, ProductName: "Strawberries", ProducerName: "Bio Hof", Country: "CH", Category: "fruit", Barcode: "123", TrustScore: 94, ColdChainScore: float64Ptr(98)},
				{Rank: 2, ProductName: "Honey", ProducerName: "Imkerei", Country: "CH", Category: "honey", Barcode: "456", TrustScore: 88},
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "empty leaderboard",
			query:      "",
			entries:    []domain.LeaderboardEntry{},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockLeaderboardRepo{entries: tt.entries}
			h := NewLeaderboardHandler(repo)

			req := httptest.NewRequest("GET", "/api/leaderboard"+tt.query, nil)
			w := httptest.NewRecorder()
			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp Response
			json.NewDecoder(w.Body).Decode(&resp)

			dataJSON, _ := json.Marshal(resp.Data)
			var lr domain.LeaderboardResponse
			json.Unmarshal(dataJSON, &lr)

			if lr.Count != tt.wantCount {
				t.Errorf("count = %d, want %d", lr.Count, tt.wantCount)
			}
			if len(lr.Leaderboard) != tt.wantCount {
				t.Errorf("entries = %d, want %d", len(lr.Leaderboard), tt.wantCount)
			}
		})
	}
}
