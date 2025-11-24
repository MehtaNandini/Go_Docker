package mlclient

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
)

// Client calls the Python ML scoring service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a configured ML client. Timeout applies per request.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// TodoPayload mirrors the ML service schema (snake_case fields).
type TodoPayload struct {
	Title           string     `json:"title"`
	Completed       bool       `json:"completed"`
	Tags            []string   `json:"tags"`
	DurationMinutes int        `json:"duration_minutes"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

type scoreRequest struct {
	Todos []TodoPayload `json:"todos"`
}

type scoreResponse struct {
	Results []struct {
		PriorityScore float64 `json:"priority_score"`
	} `json:"results"`
}

// Score sends a single todo to the ML service and returns its priority score.
func (c *Client) Score(ctx context.Context, todo TodoPayload) (float64, error) {
	if c == nil || c.baseURL == "" {
		return 0, errors.New("ml client disabled")
	}

	body, err := json.Marshal(scoreRequest{Todos: []TodoPayload{todo}})
	if err != nil {
		return 0, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/score", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call ml service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return 0, fmt.Errorf("ml service error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var sr scoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	if len(sr.Results) == 0 {
		return 0, errors.New("ml response missing results")
	}
	return sr.Results[0].PriorityScore, nil
}
