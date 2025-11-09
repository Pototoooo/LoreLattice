package chatpipline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// PluginSearch implements search functionality for chat pipeline
type PluginSearch struct {
	knowledgeBaseService interfaces.KnowledgeBaseService
	modelService         interfaces.ModelService
	config               *config.Config
	webSearchService     interfaces.WebSearchService
	tenantService        interfaces.TenantService
}

func NewPluginSearch(eventManager *EventManager,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	modelService interfaces.ModelService,
	config *config.Config,
	webSearchService interfaces.WebSearchService,
	tenantService interfaces.TenantService,
) *PluginSearch {
	res := &PluginSearch{
		knowledgeBaseService: knowledgeBaseService,
		modelService:         modelService,
		config:               config,
		webSearchService:     webSearchService,
		tenantService:        tenantService,
	}
	eventManager.Register(res)
	return res
}

// ActivationEvents returns the event types this plugin handles
func (p *PluginSearch) ActivationEvents() []types.EventType {
	return []types.EventType{types.CHUNK_SEARCH}
}

// OnEvent handles search events in the chat pipeline
func (p *PluginSearch) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	// Get knowledge base IDs list
	knowledgeBaseIDs := chatManage.KnowledgeBaseIDs
	if len(knowledgeBaseIDs) == 0 && chatManage.KnowledgeBaseID != "" {
		// Fall back to single knowledge base
		knowledgeBaseIDs = []string{chatManage.KnowledgeBaseID}
		logger.Infof(ctx, "No KnowledgeBaseIDs provided, falling back to single KB: %s", chatManage.KnowledgeBaseID)
	}

	if len(knowledgeBaseIDs) == 0 {
		logger.Errorf(ctx, "No knowledge base IDs available for search")
		return ErrSearch.WithError(nil)
	}

	logger.Infof(ctx, "Searching across %d knowledge base(s): %v", len(knowledgeBaseIDs), knowledgeBaseIDs)

	// Prepare search parameters
	searchParams := types.SearchParams{
		QueryText:        strings.TrimSpace(chatManage.RewriteQuery),
		VectorThreshold:  chatManage.VectorThreshold,
		KeywordThreshold: chatManage.KeywordThreshold,
		MatchCount:       chatManage.EmbeddingTopK,
	}
	logger.Infof(ctx, "Search parameters: %v", searchParams)

	// Parallel search across multiple knowledge bases
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allResults []*types.SearchResult

	for _, kbID := range knowledgeBaseIDs {
		wg.Add(1)
		go func(knowledgeBaseID string) {
			defer wg.Done()

			results, err := p.knowledgeBaseService.HybridSearch(ctx, knowledgeBaseID, searchParams)
			if err != nil {
				logger.Errorf(ctx, "Failed to search KB %s: %v", knowledgeBaseID, err)
				return
			}

			logger.Infof(ctx, "KB %s search results count: %d", knowledgeBaseID, len(results))

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(kbID)
	}

	wg.Wait()

	logger.Infof(ctx, "Total search results from all KBs: %d", len(allResults))
	chatManage.SearchResult = allResults

	// Add relevant results from chat history
	historyResult := p.getSearchResultFromHistory(chatManage)
	if historyResult != nil {
		logger.Infof(ctx, "Add history result, result count: %d", len(historyResult))
		chatManage.SearchResult = append(chatManage.SearchResult, historyResult...)
	}

	// Try search with processed query if different from rewrite query
	if chatManage.RewriteQuery != chatManage.ProcessedQuery {
		searchParams.QueryText = strings.TrimSpace(chatManage.ProcessedQuery)
		logger.Infof(ctx, "Searching with processed query: %s", searchParams.QueryText)

		var wg2 sync.WaitGroup
		var mu2 sync.Mutex
		var processedResults []*types.SearchResult

		for _, kbID := range knowledgeBaseIDs {
			wg2.Add(1)
			go func(knowledgeBaseID string) {
				defer wg2.Done()

				results, err := p.knowledgeBaseService.HybridSearch(ctx, knowledgeBaseID, searchParams)
				if err != nil {
					logger.Errorf(ctx, "Failed to search KB %s with processed query: %v", knowledgeBaseID, err)
					return
				}

				logger.Infof(ctx, "KB %s processed query results count: %d", knowledgeBaseID, len(results))

				mu2.Lock()
				processedResults = append(processedResults, results...)
				mu2.Unlock()
			}(kbID)
		}

		wg2.Wait()

		logger.Infof(ctx, "Total processed query results from all KBs: %d", len(processedResults))
		chatManage.SearchResult = append(chatManage.SearchResult, processedResults...)
	}

	// Perform web search if enabled and merge results with KB search results
	if chatManage.WebSearchEnabled && p.webSearchService != nil && p.tenantService != nil && chatManage.TenantID > 0 {
		// Get tenant to retrieve web search config
		tenant, err := p.tenantService.GetTenantByID(ctx, chatManage.TenantID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get tenant for web search: %v", err)
		} else if tenant != nil && tenant.WebSearchConfig != nil && tenant.WebSearchConfig.Provider != "" {
			// Perform web search in parallel with KB search (already completed)
			logger.Infof(ctx, "Performing web search with provider: %s", tenant.WebSearchConfig.Provider)
			webResults, err := p.webSearchService.Search(ctx, tenant.WebSearchConfig, chatManage.RewriteQuery)
			if err != nil {
				logger.Warnf(ctx, "Web search failed: %v", err)
			} else {
				// Convert web search results to SearchResult
				webSearchResults := convertWebSearchResults(webResults)
				logger.Infof(ctx, "Web search returned %d results", len(webSearchResults))
				// Merge web search results with KB search results
				if len(webSearchResults) > 0 {
					chatManage.SearchResult = append(chatManage.SearchResult, webSearchResults...)
					logger.Infof(ctx, "Merged web search results, total results: %d", len(chatManage.SearchResult))
				}
			}
		} else {
			logger.Warnf(ctx, "Web search enabled but no valid configuration found for tenant %d", chatManage.TenantID)
		}
	}

	// Remove duplicate results
	chatManage.SearchResult = removeDuplicateResults(chatManage.SearchResult)

	// Return if we have results
	if len(chatManage.SearchResult) != 0 {
		logger.Infof(
			ctx,
			"Get search results, count: %d, session_id: %s",
			len(chatManage.SearchResult), chatManage.SessionID,
		)
		return next()
	}
	logger.Infof(ctx, "No search result, session_id: %s", chatManage.SessionID)
	return ErrSearchNothing
}

// getSearchResultFromHistory retrieves relevant knowledge references from chat history
func (p *PluginSearch) getSearchResultFromHistory(chatManage *types.ChatManage) []*types.SearchResult {
	if len(chatManage.History) == 0 {
		return nil
	}
	// Search history in reverse chronological order
	for i := len(chatManage.History) - 1; i >= 0; i-- {
		if len(chatManage.History[i].KnowledgeReferences) > 0 {
			// Mark all references as history matches
			for _, reference := range chatManage.History[i].KnowledgeReferences {
				reference.MatchType = types.MatchTypeHistory
			}
			return chatManage.History[i].KnowledgeReferences
		}
	}
	return nil
}

func removeDuplicateResults(results []*types.SearchResult) []*types.SearchResult {
	seen := make(map[string]bool)
	var uniqueResults []*types.SearchResult
	for _, result := range results {
		if !seen[result.ID] {
			seen[result.ID] = true
			uniqueResults = append(uniqueResults, result)
		}
	}
	return uniqueResults
}

// convertWebSearchResults converts WebSearchResult to SearchResult
// This is a duplicate of the function in service/web_search.go to avoid circular imports
func convertWebSearchResults(webResults []*types.WebSearchResult) []*types.SearchResult {
	results := make([]*types.SearchResult, 0, len(webResults))

	for i, webResult := range webResults {
		// Use URL as ChunkID for web search results
		chunkID := webResult.URL
		if chunkID == "" {
			chunkID = fmt.Sprintf("web_search_%d", i)
		}

		// Combine title and snippet as content
		content := webResult.Title
		if webResult.Snippet != "" {
			if content != "" {
				content += "\n\n" + webResult.Snippet
			} else {
				content = webResult.Snippet
			}
		}
		if webResult.Content != "" {
			if content != "" {
				content += "\n\n" + webResult.Content
			} else {
				content = webResult.Content
			}
		}

		// Set a default score for web search results (0.6, indicating medium relevance)
		score := 0.6

		result := &types.SearchResult{
			ID:             chunkID,
			Content:        content,
			KnowledgeID:    "", // Web search results don't have knowledge ID
			ChunkIndex:     0,
			KnowledgeTitle: webResult.Title,
			StartAt:        0,
			EndAt:          len(content),
			Seq:            i,
			Score:          score,
			MatchType:      types.MatchTypeWebSearch,
			SubChunkID:     []string{},
			Metadata: map[string]string{
				"url":     webResult.URL,
				"source":  webResult.Source,
				"title":   webResult.Title,
				"snippet": webResult.Snippet,
			},
			ChunkType:         "web_search",
			ParentChunkID:     "",
			ImageInfo:         "",
			KnowledgeFilename: "",
			KnowledgeSource:   "web_search",
		}

		// Add published date to metadata if available
		if webResult.PublishedAt != nil {
			result.Metadata["published_at"] = webResult.PublishedAt.Format(time.RFC3339)
		}

		results = append(results, result)
	}

	return results
}
