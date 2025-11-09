package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// SearxngProvider implements web search using Searxng API
type SearxngProvider struct {
	apiURL string
}

// NewSearxngProvider creates a new Searxng provider
func NewSearxngProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &SearxngProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *SearxngProvider) Name() string {
	return "searxng"
}

// Search performs a web search using Searxng API
func (p *SearxngProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Searxng search API
	return nil, fmt.Errorf("searxng search provider is not yet implemented")
}

