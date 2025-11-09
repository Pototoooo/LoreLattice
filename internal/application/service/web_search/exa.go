package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// ExaProvider implements web search using Exa API
type ExaProvider struct {
	apiKey string
	apiURL string
}

// NewExaProvider creates a new Exa provider
func NewExaProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &ExaProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *ExaProvider) Name() string {
	return "exa"
}

// Search performs a web search using Exa API
func (p *ExaProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Exa search API
	return nil, fmt.Errorf("exa search provider is not yet implemented")
}

// SetAPIKey sets the API key for the provider
func (p *ExaProvider) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}
