package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// KuaisouProvider implements web search using Kuaisou API
type KuaisouProvider struct {
	apiKey string
	apiURL string
	client *http.Client
}

// NewKuaisouProvider creates a new Kuaisou provider
func NewKuaisouProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("kuaisou API URL is required")
	}

	return &KuaisouProvider{
		apiKey: "", // Will be set from tenant config
		apiURL: cfg.APIURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *KuaisouProvider) Name() string {
	return "kuaisou"
}

// Search performs a web search using Kuaisou API
func (p *KuaisouProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("kuaisou API key is required")
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"query":       query,
		"max_results": maxResults,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kuaisou API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to WebSearchResult
	results := make([]*types.WebSearchResult, 0, len(response.Results))
	for _, item := range response.Results {
		results = append(results, &types.WebSearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Snippet,
			Source:  "kuaisou",
		})
	}

	logger.Infof(ctx, "Kuaisou search returned %d results for query: %s", len(results), query)
	return results, nil
}

// SetAPIKey sets the API key for the provider
func (p *KuaisouProvider) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}

