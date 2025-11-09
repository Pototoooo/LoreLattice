package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// GoogleProvider implements web search using Google Custom Search API
type GoogleProvider struct {
	apiKey        string
	searchEngineID string
	apiURL        string
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &GoogleProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// Search performs a web search using Google Custom Search API
func (p *GoogleProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Google Custom Search API
	return nil, fmt.Errorf("google search provider is not yet implemented")
}

// SetAPIKey sets the API key and search engine ID
func (p *GoogleProvider) SetAPIKey(apiKey, searchEngineID string) {
	p.apiKey = apiKey
	p.searchEngineID = searchEngineID
}

