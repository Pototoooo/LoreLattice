package chatpipline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/redis/go-redis/v9"
)

// PluginSearch implements search functionality for chat pipeline
type PluginSearch struct {
	knowledgeBaseService interfaces.KnowledgeBaseService
	knowledgeService     interfaces.KnowledgeService
	modelService         interfaces.ModelService
	config               *config.Config
	webSearchService     interfaces.WebSearchService
	tenantService        interfaces.TenantService
	redisClient          *redis.Client
}

func NewPluginSearch(eventManager *EventManager,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	modelService interfaces.ModelService,
	config *config.Config,
	webSearchService interfaces.WebSearchService,
	tenantService interfaces.TenantService,
	redisClient *redis.Client,
) *PluginSearch {
	res := &PluginSearch{
		knowledgeBaseService: knowledgeBaseService,
		knowledgeService:     knowledgeService,
		modelService:         modelService,
		config:               config,
		webSearchService:     webSearchService,
		tenantService:        tenantService,
		redisClient:          redisClient,
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

	// Run KB search and web search concurrently
	logger.Infof(ctx, "Searching across %d knowledge base(s): %v", len(knowledgeBaseIDs), knowledgeBaseIDs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allResults := make([]*types.SearchResult, 0)

	wg.Add(2)
	// Goroutine 1: Knowledge base search (rewrite + processed)
	go func() {
		defer wg.Done()
		kbResults := p.searchKnowledgeBases(ctx, knowledgeBaseIDs, chatManage)
		if len(kbResults) > 0 {
			mu.Lock()
			allResults = append(allResults, kbResults...)
			mu.Unlock()
		}
	}()

	// Goroutine 2: Web search (if enabled)
	go func() {
		defer wg.Done()
		webResults := p.searchWebIfEnabled(ctx, chatManage)
		if len(webResults) > 0 {
			mu.Lock()
			allResults = append(allResults, webResults...)
			mu.Unlock()
		}
	}()

	wg.Wait()

	chatManage.SearchResult = allResults

	// Add relevant results from chat history
	historyResult := p.getSearchResultFromHistory(chatManage)
	if historyResult != nil {
		logger.Infof(ctx, "Add history result, result count: %d", len(historyResult))
		chatManage.SearchResult = append(chatManage.SearchResult, historyResult...)
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

// searchKnowledgeBases performs KB searches for rewrite and processed queries across KB IDs
func (p *PluginSearch) searchKnowledgeBases(ctx context.Context, knowledgeBaseIDs []string, chatManage *types.ChatManage) []*types.SearchResult {
	// Build base params for rewrite query
	baseParams := types.SearchParams{
		QueryText:        strings.TrimSpace(chatManage.RewriteQuery),
		VectorThreshold:  chatManage.VectorThreshold,
		KeywordThreshold: chatManage.KeywordThreshold,
		MatchCount:       chatManage.EmbeddingTopK,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []*types.SearchResult

	// Search with rewrite query
	for _, kbID := range knowledgeBaseIDs {
		wg.Add(1)
		go func(knowledgeBaseID string) {
			defer wg.Done()
			res, err := p.knowledgeBaseService.HybridSearch(ctx, knowledgeBaseID, baseParams)
			if err != nil {
				logger.Errorf(ctx, "Failed to search KB %s: %v", knowledgeBaseID, err)
				return
			}
			logger.Infof(ctx, "KB %s search results count: %d", knowledgeBaseID, len(res))
			mu.Lock()
			results = append(results, res...)
			mu.Unlock()
		}(kbID)
	}

	wg.Wait()

	// If processed query differs, search again
	if chatManage.RewriteQuery != chatManage.ProcessedQuery {
		paramsProcessed := baseParams
		paramsProcessed.QueryText = strings.TrimSpace(chatManage.ProcessedQuery)
		logger.Infof(ctx, "Searching with processed query: %s", paramsProcessed.QueryText)

		wg = sync.WaitGroup{}
		for _, kbID := range knowledgeBaseIDs {
			wg.Add(1)
			go func(knowledgeBaseID string) {
				defer wg.Done()
				res, err := p.knowledgeBaseService.HybridSearch(ctx, knowledgeBaseID, paramsProcessed)
				if err != nil {
					logger.Errorf(ctx, "Failed to search KB %s with processed query: %v", knowledgeBaseID, err)
					return
				}
				logger.Infof(ctx, "KB %s processed query results count: %d", knowledgeBaseID, len(res))
				mu.Lock()
				results = append(results, res...)
				mu.Unlock()
			}(kbID)
		}
		wg.Wait()
	}

	logger.Infof(ctx, "Total KB results (rewrite + processed): %d", len(results))
	return results
}

// searchWebIfEnabled executes web search when enabled and returns converted results
func (p *PluginSearch) searchWebIfEnabled(ctx context.Context, chatManage *types.ChatManage) []*types.SearchResult {
	if !(chatManage.WebSearchEnabled && p.webSearchService != nil && p.tenantService != nil && chatManage.TenantID > 0) {
		return nil
	}
	tenant := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if tenant == nil || tenant.WebSearchConfig == nil || tenant.WebSearchConfig.Provider == "" {
		logger.Warnf(ctx, "Web search enabled but no valid configuration found for tenant %d", chatManage.TenantID)
		return nil
	}

	logger.Infof(ctx, "Performing web search with provider: %s", tenant.WebSearchConfig.Provider)
	webResults, err := p.webSearchService.Search(ctx, tenant.WebSearchConfig, chatManage.RewriteQuery)
	if err != nil {
		logger.Warnf(ctx, "Web search failed: %v", err)
		return nil
	}
	// Build questions (rewrite + processed if different)
	questions := []string{strings.TrimSpace(chatManage.RewriteQuery)}
	if chatManage.ProcessedQuery != "" && chatManage.ProcessedQuery != chatManage.RewriteQuery {
		questions = append(questions, strings.TrimSpace(chatManage.ProcessedQuery))
	}
	// Load session-scoped temp KB state from Redis
	var tempKBID string
	seen := map[string]bool{}
	ids := []string{}
	stateKey := fmt.Sprintf("tempkb:%s", chatManage.SessionID)
	if raw, getErr := p.redisClient.Get(ctx, stateKey).Bytes(); getErr == nil && len(raw) > 0 {
		var state struct {
			KBID         string          `json:"kbID"`
			KnowledgeIDs []string        `json:"knowledgeIDs"`
			SeenURLs     map[string]bool `json:"seenURLs"`
		}
		if err := json.Unmarshal(raw, &state); err == nil {
			tempKBID = state.KBID
			ids = state.KnowledgeIDs
			if state.SeenURLs != nil {
				seen = state.SeenURLs
			}
		}
	}
	compressed, kbID, newSeen, newIDs, err := p.webSearchService.CompressWithRAG(
		ctx, chatManage.SessionID, tempKBID, questions, webResults, tenant.WebSearchConfig,
		p.knowledgeBaseService, p.knowledgeService, seen, ids,
	)
	if err != nil {
		logger.Warnf(ctx, "RAG compression failed, falling back to raw: %v", err)
	} else {
		webResults = compressed
		// Persist temp KB state back into Redis
		state := struct {
			KBID         string          `json:"kbID"`
			KnowledgeIDs []string        `json:"knowledgeIDs"`
			SeenURLs     map[string]bool `json:"seenURLs"`
		}{
			KBID:         kbID,
			KnowledgeIDs: newIDs,
			SeenURLs:     newSeen,
		}
		if b, mErr := json.Marshal(state); mErr == nil {
			_ = p.redisClient.Set(ctx, stateKey, b, 0).Err()
		}
	}
	res := convertWebSearchResults(webResults)
	logger.Infof(ctx, "Web search returned %d results", len(res))
	return res
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
