package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// ZhipuProvider implements web search using Zhipu API
type ZhipuProvider struct {
	apiKey string
	apiURL string
}

// NewZhipuProvider creates a new Zhipu provider
func NewZhipuProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &ZhipuProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *ZhipuProvider) Name() string {
	return "zhipu"
}

// Search performs a web search using Zhipu API
func (p *ZhipuProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Zhipu search API
	return nil, fmt.Errorf("zhipu search provider is not yet implemented")
}

// SetAPIKey sets the API key for the provider
func (p *ZhipuProvider) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}
