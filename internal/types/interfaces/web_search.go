package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// WebSearchProvider defines the interface for web search providers
type WebSearchProvider interface {
	Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error)
	Name() string
}

type WebSearchService interface {
	Search(ctx context.Context, config *types.WebSearchConfig, query string) ([]*types.WebSearchResult, error)
	CompressWithRAG(ctx context.Context, sessionID string, tempKBID string, questions []string,
		webSearchResults []*types.WebSearchResult, cfg *types.WebSearchConfig,
		kbSvc KnowledgeBaseService, knowSvc KnowledgeService,
		seenURLs map[string]bool, knowledgeIDs []string,
	) (compressed []*types.WebSearchResult, kbID string, newSeen map[string]bool, newIDs []string, err error)
}
