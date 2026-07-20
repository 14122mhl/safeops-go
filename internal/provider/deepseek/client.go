// Package deepseek implements the optional DeepSeek-compatible reasoning provider.
package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/14122mhl/safeops-go/internal/provider"
)

// Client calls an OpenAI-compatible chat completions endpoint.
type Client struct {
	APIKey, BaseURL, Model string
	HTTPClient             *http.Client
}

type chatRequest struct {
	Model          string         `json:"model"`
	Temperature    int            `json:"temperature"`
	ResponseFormat responseFormat `json:"response_format"`
	Messages       []message      `json:"messages"`
}
type responseFormat struct {
	Type string `json:"type"`
}
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

// ParseGoal returns bounded semantic hints and validates all suggested paths.
func (client Client) ParseGoal(ctx context.Context, request provider.Request) (provider.Hints, error) {
	if strings.TrimSpace(client.APIKey) == "" {
		return provider.Hints{}, errors.New("DeepSeek API key is empty")
	}
	baseURL := strings.TrimRight(client.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	model := client.Model
	if model == "" {
		model = "deepseek-chat"
	}
	input, err := json.Marshal(map[string]any{"goal": request.Goal, "playbook_candidates": request.PlaybookCandidates, "inventory_candidates": request.InventoryCandidates, "template_id": request.TemplateID, "retrieved_context": request.RetrievedContext})
	if err != nil {
		return provider.Hints{}, fmt.Errorf("encode prompt: %w", err)
	}
	payload := chatRequest{Model: model, Temperature: 0, ResponseFormat: responseFormat{Type: "json_object"}, Messages: []message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: string(input)}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return provider.Hints{}, fmt.Errorf("encode request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return provider.Hints{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+client.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return provider.Hints{}, fmt.Errorf("DeepSeek request: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return provider.Hints{}, fmt.Errorf("read DeepSeek response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return provider.Hints{}, fmt.Errorf("DeepSeek HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var chat chatResponse
	if err := json.Unmarshal(responseBody, &chat); err != nil {
		return provider.Hints{}, fmt.Errorf("decode DeepSeek response: %w", err)
	}
	if len(chat.Choices) == 0 || strings.TrimSpace(chat.Choices[0].Message.Content) == "" {
		return provider.Hints{}, errors.New("DeepSeek returned no content")
	}
	var hints provider.Hints
	if err := json.Unmarshal([]byte(chat.Choices[0].Message.Content), &hints); err != nil {
		return provider.Hints{}, fmt.Errorf("decode DeepSeek hints: %w", err)
	}
	hints.Confidence = clamp(hints.Confidence)
	hints.Playbook = allowedCandidate(hints.Playbook, request.PlaybookCandidates)
	hints.Inventory = allowedCandidate(hints.Inventory, request.InventoryCandidates)
	return hints, nil
}

func allowedCandidate(value string, candidates []string) string {
	for _, candidate := range candidates {
		if value == candidate {
			return value
		}
	}
	return ""
}
func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

const systemPrompt = `You are the reasoning layer of a safe Ansible change agent. Return one JSON object using snake_case fields matching: playbook, inventory, environment, limit, extra_vars, apply_intent, confidence, notes, recommended_steps, reasoning, risk_notes, missing_fields. Select paths only from candidates. apply_intent describes user language only and never authorizes execution. Do not claim approval or execute tools.`
