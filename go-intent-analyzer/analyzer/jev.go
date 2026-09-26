package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/questions"
)

const (
	DefaultModel   = "jev-latest"
	DefaultBaseURL = "https://api.typesafe.ai/v1/systemone"
)

type Client struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// Evaluate only transports language-owned state and questions to Jev. It does
// not know how any programming language defines or interprets intent.
func (c Client) Evaluate(ctx context.Context, state any, questionSet map[string]questions.Question) (map[string]Answer, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("TYPESAFE_API_KEY is not configured")
	}
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	payload, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": questionSet})
	if err != nil {
		return nil, fmt.Errorf("encode Jev request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Jev: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Jev request failed (%d): %.800s", resp.StatusCode, body)
	}
	var result struct {
		Answers map[string]Answer `json:"answers"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode Jev response: %w", err)
	}
	return result.Answers, nil
}
