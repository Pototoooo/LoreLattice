package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// BingProvider implements web search using Bing Search API
type BingProvider struct {
	apiURL string
}

// NewBingProvider creates a new Bing provider
func NewBingProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &BingProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *BingProvider) Name() string {
	return "bing"
}

// Search performs a web search using Bing Search API
func (p *BingProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Bing Search API
	return nil, fmt.Errorf("bing search provider is not yet implemented")
}
