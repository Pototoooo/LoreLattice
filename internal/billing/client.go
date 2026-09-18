package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("MeterForge returned HTTP %d: %s", e.Status, e.Body)
}

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) SendEvent(ctx context.Context, eventID string, payload json.RawMessage) error {
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}
	var reader io.Reader
	reader = bytes.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/events", reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/cloudevents+json")
	req.Header.Set("Idempotency-Key", eventID)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	return nil
}
