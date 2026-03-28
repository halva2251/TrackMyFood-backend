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
	groqKey   string
	geminiKey string
}

func NewChatService(groqKey, geminiKey string) *ChatService {
	return &ChatService{groqKey: groqKey, geminiKey: geminiKey}
}

func (s *ChatService) Ask(ctx context.Context, scanData *domain.ScanResponse, question string) (string, error) {
	if s.groqKey == "" && s.geminiKey == "" {
		return "I'm sorry, I'm currently in a manual mode because my AI brain (API key) hasn't been configured yet. But looking at the data, the Trust Score is " + fmt.Sprintf("%.0f", scanData.TrustScore.Overall) + ".", nil
	}

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

	// Try Groq first (free, fast), fall back to Gemini
	if s.groqKey != "" {
		answer, err := s.askGroq(ctx, prompt)
		if err == nil {
			return answer, nil
		}
		// If Groq fails and we have Gemini, fall through
		if s.geminiKey == "" {
			return "", fmt.Errorf("groq api error: %w", err)
		}
	}

	return s.askGemini(ctx, prompt)
}

// askGroq uses the Groq API (OpenAI-compatible).
func (s *ChatService) askGroq(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model": "llama-3.1-8b-instant",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  1024,
		"temperature": 0.7,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.groqKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("status %d: %v", resp.StatusCode, errResp)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no choices returned")
}

// askGemini uses the Google Gemini API.
func (s *ChatService) askGemini(ctx context.Context, prompt string) (string, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	reqBody := struct {
		Contents []content `json:"contents"`
	}{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", s.geminiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("gemini api error (status %d): %v", resp.StatusCode, errResp)
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("no response from gemini")
}
