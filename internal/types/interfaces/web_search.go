package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// WebSearchService defines the interface for web search service
type WebSearchService interface {
	Search(ctx context.Context, config *types.WebSearchConfig, query string) ([]*types.WebSearchResult, error)
}
