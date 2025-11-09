package service

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
)

// BaiduProvider implements web search using Baidu API
type BaiduProvider struct {
	apiURL string
}

// NewBaiduProvider creates a new Baidu provider
func NewBaiduProvider(cfg config.WebSearchProviderConfig) (WebSearchProvider, error) {
	return &BaiduProvider{
		apiURL: cfg.APIURL,
	}, nil
}

// Name returns the provider name
func (p *BaiduProvider) Name() string {
	return "baidu"
}

// Search performs a web search using Baidu API
func (p *BaiduProvider) Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error) {
	// TODO: Implement Baidu search API
	return nil, fmt.Errorf("baidu search provider is not yet implemented")
}

