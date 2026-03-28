package service

import (
	"context"
	"strings"
	"testing"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

func TestChatService_Ask_Fallback(t *testing.T) {
	// Test without an API key to ensure the fallback logic works
	svc := NewChatService("", "")
	scanData := &domain.ScanResponse{
		Product: domain.ScanProduct{Name: "Mate"},
		TrustScore: domain.ScanTrustScore{Overall: 88},
	}

	answer, err := svc.Ask(context.Background(), scanData, "How is it?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(answer, "88") {
		t.Errorf("expected answer to contain score '88', got: %s", answer)
	}

	if !strings.Contains(answer, "manual mode") {
		t.Errorf("expected answer to mention 'manual mode', got: %s", answer)
	}
}
