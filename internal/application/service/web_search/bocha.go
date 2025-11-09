package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// BochaProvider implements web search using Bocha API
type BochaProvider struct {
	apiKey string
	apiURL string
}

// NewBochaProvider creates a new Bocha provider
func NewBochaProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &BochaProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *BochaProvider) Name() string {
	return "bocha"
}

// Search performs a web search using Bocha API
func (p *BochaProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Bocha search API
	return nil, fmt.Errorf("bocha search provider is not yet implemented")
}

// SetAPIKey sets the API key for the provider
func (p *BochaProvider) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}
