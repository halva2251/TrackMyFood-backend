package domain_test

import (
	"testing"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

func TestTrustScoreLabel(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{100, "Excellent"},
		{94, "Excellent"},
		{80, "Excellent"},
		{79.99, "Good"},
		{60, "Good"},
		{59.99, "Fair"},
		{40, "Fair"},
		{39.99, "Poor"},
		{20, "Poor"},
		{19.99, "Critical"},
		{0, "Critical"},
	}
	for _, tt := range tests {
		got := domain.TrustScoreLabel(tt.score)
		if got != tt.want {
			t.Errorf("TrustScoreLabel(%v) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestTrustScoreColor(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{100, "green"},
		{60, "green"},
		{59.99, "orange"},
		{40, "orange"},
		{39.99, "red"},
		{0, "red"},
	}
	for _, tt := range tests {
		got := domain.TrustScoreColor(tt.score)
		if got != tt.want {
			t.Errorf("TrustScoreColor(%v) = %q, want %q", tt.score, got, tt.want)
		}
	}
}
