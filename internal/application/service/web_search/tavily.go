package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// TavilyProvider implements web search using Tavily API
type TavilyProvider struct {
	apiKey string
	apiURL string
}

// NewTavilyProvider creates a new Tavily provider
func NewTavilyProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &TavilyProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *TavilyProvider) Name() string {
	return "tavily"
}

// Search performs a web search using Tavily API
func (p *TavilyProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Tavily search API
	return nil, fmt.Errorf("tavily search provider is not yet implemented")
}

// SetAPIKey sets the API key for the provider
func (p *TavilyProvider) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}
