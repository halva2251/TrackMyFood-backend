package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type LeaderboardGetter interface {
	GetTopBatches(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error)
}

type LeaderboardHandler struct {
	repo LeaderboardGetter
}

func NewLeaderboardHandler(repo LeaderboardGetter) *LeaderboardHandler {
	return &LeaderboardHandler{repo: repo}
}

func (h *LeaderboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := h.repo.GetTopBatches(r.Context(), limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to get leaderboard")
		return
	}

	WriteJSON(w, http.StatusOK, domain.LeaderboardResponse{
		Leaderboard: entries,
		Count:       len(entries),
	})
}
