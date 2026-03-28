package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
)

type ChatService struct {
	apiKey string
}

func NewChatService(apiKey string) *ChatService {
	return &ChatService{apiKey: apiKey}
}

type geminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (s *ChatService) Ask(ctx context.Context, scanData *domain.ScanResponse, question string) (string, error) {
	if s.apiKey == "" {
		return "I'm sorry, I'm currently in a manual mode because my AI brain (API key) hasn't been configured yet. But looking at the data, the Trust Score is " + fmt.Sprintf("%.0f", scanData.TrustScore.Overall) + ".", nil
	}

	// Build the context-rich prompt
	scanJSON, _ := json.MarshalIndent(scanData, "", "  ")
	prompt := fmt.Sprintf(`You are "Foodie", the AI assistant for "Track My Food".
Your goal is to answer questions about a specific food product batch based on the provided tracking data.
Be helpful, transparent, and focus on food safety and supply chain integrity.

PRODUCT DATA:
%s

USER QUESTION:
%s

ANSWER GUIDELINES:
- If there's a recall, mention it immediately as the highest priority.
- If the Trust Score is low, explain why (e.g., cold chain breaches, failed checks).
- If the question is about sustainability, use the CO2 and certification data.
- Keep answers concise and friendly.
`, string(scanJSON), question)

	reqBody := geminiRequest{}
	reqBody.Contents = append(reqBody.Contents, struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		Parts: []struct {
			Text string `json:"text"`
		}{
			{Text: prompt},
		},
	})

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-8b:generateContent?key=%s", s.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("gemini api error (status %d): %v", resp.StatusCode, errResp)
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "I analyzed the data but couldn't formulate a specific answer. The product seems to be in " + scanData.TrustScore.Label + " condition.", nil
}
