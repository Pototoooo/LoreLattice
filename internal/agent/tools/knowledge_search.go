package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// searchResultWithMeta wraps search result with metadata about which query matched it
type searchResultWithMeta struct {
	*types.SearchResult
	SourceQuery string
	QueryType   string // "vector" or "keyword"
}

// KnowledgeSearchTool searches knowledge bases with flexible query modes
type KnowledgeSearchTool struct {
	BaseTool
	knowledgeService interfaces.KnowledgeBaseService
	tenantID         uint
	allowedKBs       []string
	rerankModel      rerank.Reranker
}

// NewKnowledgeSearchTool creates a new knowledge search tool
func NewKnowledgeSearchTool(
	knowledgeService interfaces.KnowledgeBaseService,
	tenantID uint,
	allowedKBs []string,
	rerankModel rerank.Reranker,
) *KnowledgeSearchTool {
	description := `Search within knowledge bases with flexible query modes. Unified tool that supports both targeted and broad searches.

## Features
- Multi-KB search: Search across multiple knowledge bases concurrently
- Flexible queries: Support vector, keyword, or hybrid search modes
- Quality filtering: Automatically filters low-quality chunks

## Usage

**Use when**:
- You know which knowledge bases to target (specify knowledge_base_ids)
- You're unsure which KB contains the info (omit knowledge_base_ids to search all allowed KBs)
- Want to search specific KBs with same query
- Need semantic (vector) or exact keyword searches
- Want to search only specific documents within KBs

**Parameters**:
- knowledge_base_ids (optional): Array of KB IDs to search (1-10). If omitted, searches all allowed KBs.
- query (optional): Single search query (for simple hybrid search)
- vector_queries (optional): Array of semantic queries for vector search (1-5 queries)
- keyword_queries (optional): Array of keyword queries for keyword search (1-5 queries)
- top_k (optional): Results per KB per query (default: 5, max: 20)
- vector_threshold (optional): Minimum score for vector results (default: 0.6, 0.0-1.0)
- keyword_threshold (optional): Minimum score for keyword results (default: 0.5, 0.0-1.0)
- knowledge_ids (optional): Array of document IDs to filter results (only return results from these documents)
- min_score (optional): Absolute minimum score to include results (default: 0.3, filters very low quality chunks)

**Search Modes**:
- Simple: Provide single query parameter (hybrid search)
- Vector only: Provide vector_queries only
- Keyword only: Provide keyword_queries only
- Hybrid: Provide both vector_queries and keyword_queries
- At least one query parameter must be provided

**Returns**: Merged and deduplicated search results from all KBs

## Examples

` + "`" + `
# Simple search in specific KBs
{
  "knowledge_base_ids": ["kb1", "kb2"],
  "query": "什么是向量数据库"
}

# Search all allowed KBs with vector queries
{
  "vector_queries": ["什么是向量数据库", "向量数据库的定义"]
}

# Multiple query types with thresholds
{
  "knowledge_base_ids": ["kb1"],
  "vector_queries": ["向量数据库应用"],
  "keyword_queries": ["Docker", "部署"],
  "vector_threshold": 0.7,
  "keyword_threshold": 0.6
}

# Search specific documents
{
  "knowledge_base_ids": ["kb1"],
  "query": "彗星的起源",
  "knowledge_ids": ["doc1", "doc2"]
}
` + "`" + `

## Tips

- Concurrent search across multiple KBs and queries
- Results are automatically reranked to unify scores from different sources
- Reranked scores are in 0-1 range and directly comparable
- Results are merged, deduplicated and sorted by relevance
- Use vector_queries for semantic/conceptual searches
- Use keyword_queries for exact term matching
- Results below threshold are automatically filtered
- High relevance (>=0.8): directly usable
- Medium relevance (0.6-0.8): reference only
- Low relevance (<0.6): use with caution`

	return &KnowledgeSearchTool{
		BaseTool:         NewBaseTool("knowledge_search", description),
		knowledgeService: knowledgeService,
		tenantID:         tenantID,
		allowedKBs:       allowedKBs,
		rerankModel:      rerankModel,
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *KnowledgeSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"knowledge_base_ids": map[string]interface{}{
				"type":        "array",
				"description": "Array of knowledge base IDs to search in (optional, if omitted searches all allowed KBs)",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 10,
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Single search query for simple hybrid search",
			},
			"vector_queries": map[string]interface{}{
				"type":        "array",
				"description": "Array of semantic queries for vector search (1-5 queries)",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 5,
			},
			"keyword_queries": map[string]interface{}{
				"type":        "array",
				"description": "Array of keyword queries for keyword search (1-5 queries)",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 5,
			},
			"top_k": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results per knowledge base per query (default: 5)",
				"default":     5,
				"minimum":     1,
				"maximum":     20,
			},
			"vector_threshold": map[string]interface{}{
				"type":        "number",
				"description": "Minimum score for vector results (default: 0.6)",
				"default":     0.6,
				"minimum":     0.0,
				"maximum":     1.0,
			},
			"keyword_threshold": map[string]interface{}{
				"type":        "number",
				"description": "Minimum score for keyword results (default: 0.5)",
				"default":     0.5,
				"minimum":     0.0,
				"maximum":     1.0,
			},
			"knowledge_ids": map[string]interface{}{
				"type":        "array",
				"description": "Optional array of document IDs to filter results (only return results from these specific documents)",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 50,
			},
			"min_score": map[string]interface{}{
				"type":        "number",
				"description": "Absolute minimum score threshold for filtering very low quality results (default: 0.3)",
				"default":     0.3,
				"minimum":     0.0,
				"maximum":     1.0,
			},
		},
		"required": []string{},
	}
}

// Execute executes the knowledge search tool with flexible query modes
func (t *KnowledgeSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][KnowledgeSearch] Execute started")

	// Log input arguments
	argsJSON, _ := json.MarshalIndent(args, "", "  ")
	logger.Debugf(ctx, "[Tool][KnowledgeSearch] Input args:\n%s", string(argsJSON))

	// Determine which KBs to search
	var kbIDs []string
	if kbIDsRaw, ok := args["knowledge_base_ids"].([]interface{}); ok && len(kbIDsRaw) > 0 {
		for _, id := range kbIDsRaw {
			if idStr, ok := id.(string); ok && idStr != "" {
				kbIDs = append(kbIDs, idStr)
			}
		}
		logger.Infof(ctx, "[Tool][KnowledgeSearch] User specified %d knowledge bases: %v", len(kbIDs), kbIDs)
	}

	// If no KBs specified, use allowed KBs
	if len(kbIDs) == 0 {
		kbIDs = t.allowedKBs
		if len(kbIDs) == 0 {
			logger.Errorf(ctx, "[Tool][KnowledgeSearch] No knowledge bases available")
			return &types.ToolResult{
				Success: false,
				Error:   "no knowledge bases specified and no allowed KBs configured",
			}, fmt.Errorf("no knowledge bases available")
		}
		logger.Infof(ctx, "[Tool][KnowledgeSearch] Using all allowed KBs (%d): %v", len(kbIDs), kbIDs)
	}

	// Parse query parameters
	var singleQuery string
	var vectorQueries, keywordQueries []string

	// Parse single query
	if q, ok := args["query"].(string); ok && q != "" {
		singleQuery = q
	}

	// Parse vector_queries
	if vq, ok := args["vector_queries"].([]interface{}); ok {
		for _, q := range vq {
			if queryStr, ok := q.(string); ok && queryStr != "" {
				vectorQueries = append(vectorQueries, queryStr)
			}
		}
	}

	// Parse keyword_queries
	if kq, ok := args["keyword_queries"].([]interface{}); ok {
		for _, q := range kq {
			if queryStr, ok := q.(string); ok && queryStr != "" {
				keywordQueries = append(keywordQueries, queryStr)
			}
		}
	}

	// If single query provided, treat it as both vector and keyword query
	if singleQuery != "" {
		if len(vectorQueries) == 0 && len(keywordQueries) == 0 {
			vectorQueries = []string{singleQuery}
			keywordQueries = []string{singleQuery}
		}
	}

	// Validate: at least one query must be provided
	if len(vectorQueries) == 0 && len(keywordQueries) == 0 {
		logger.Errorf(ctx, "[Tool][KnowledgeSearch] No query provided")
		return &types.ToolResult{
			Success: false,
			Error:   "at least one of query, vector_queries, or keyword_queries must be provided",
		}, fmt.Errorf("no query provided")
	}

	logger.Infof(ctx, "[Tool][KnowledgeSearch] Query mode: single=%v, vector_queries=%d, keyword_queries=%d",
		singleQuery != "", len(vectorQueries), len(keywordQueries))
	if singleQuery != "" {
		logger.Debugf(ctx, "[Tool][KnowledgeSearch] Single query: %s", singleQuery)
	}
	if len(vectorQueries) > 0 {
		logger.Debugf(ctx, "[Tool][KnowledgeSearch] Vector queries: %v", vectorQueries)
	}
	if len(keywordQueries) > 0 {
		logger.Debugf(ctx, "[Tool][KnowledgeSearch] Keyword queries: %v", keywordQueries)
	}

	// Parse thresholds
	vectorThreshold := 0.6
	if vt, ok := args["vector_threshold"].(float64); ok {
		vectorThreshold = vt
	}

	keywordThreshold := 0.5
	if kt, ok := args["keyword_threshold"].(float64); ok {
		keywordThreshold = kt
	}

	// Parse min_score for absolute filtering
	minScore := 0.3
	if ms, ok := args["min_score"].(float64); ok {
		minScore = ms
	}

	// Parse top_k
	topK := 5
	if topKVal, ok := args["top_k"]; ok {
		switch v := topKVal.(type) {
		case float64:
			topK = int(v)
		case int:
			topK = v
		}
	}

	logger.Infof(ctx, "[Tool][KnowledgeSearch] Search params: top_k=%d, vector_threshold=%.2f, keyword_threshold=%.2f, min_score=%.2f",
		topK, vectorThreshold, keywordThreshold, minScore)

	// Extract knowledge_ids filter if provided
	var knowledgeIDsFilter map[string]bool
	if knowledgeIDsRaw, ok := args["knowledge_ids"].([]interface{}); ok && len(knowledgeIDsRaw) > 0 {
		knowledgeIDsFilter = make(map[string]bool)
		for _, id := range knowledgeIDsRaw {
			if idStr, ok := id.(string); ok && idStr != "" {
				knowledgeIDsFilter[idStr] = true
			}
		}
	}

	// Execute concurrent search
	logger.Infof(ctx, "[Tool][KnowledgeSearch] Starting concurrent search across %d KBs", len(kbIDs))
	allResults := t.concurrentSearch(ctx, vectorQueries, keywordQueries, kbIDs,
		topK, vectorThreshold, keywordThreshold)
	logger.Infof(ctx, "[Tool][KnowledgeSearch] Concurrent search completed: %d raw results", len(allResults))

	// Filter by knowledge_ids if provided
	if len(knowledgeIDsFilter) > 0 {
		logger.Infof(ctx, "[Tool][KnowledgeSearch] Filtering by %d knowledge IDs", len(knowledgeIDsFilter))
		filtered := make([]*searchResultWithMeta, 0)
		for _, r := range allResults {
			if knowledgeIDsFilter[r.KnowledgeID] {
				filtered = append(filtered, r)
			}
		}
		logger.Infof(ctx, "[Tool][KnowledgeSearch] After knowledge_id filter: %d results (from %d)",
			len(filtered), len(allResults))
		allResults = filtered
	}

	// Filter by threshold first
	logger.Infof(ctx, "[Tool][KnowledgeSearch] Applying threshold filter...")
	filteredResults := t.filterByThreshold(allResults, vectorThreshold, keywordThreshold)
	logger.Infof(ctx, "[Tool][KnowledgeSearch] After threshold filter: %d results (from %d)",
		len(filteredResults), len(allResults))

	// Apply ReRank if model is configured
	if t.rerankModel != nil && len(filteredResults) > 0 {
		logger.Infof(ctx, "[Tool][KnowledgeSearch] Applying rerank with model: %s, input: %d results",
			t.rerankModel.GetModelName(), len(filteredResults))
		rerankQuery := singleQuery
		if rerankQuery == "" && len(vectorQueries) > 0 {
			rerankQuery = vectorQueries[0] // Use first vector query as rerank query
		} else if rerankQuery == "" && len(keywordQueries) > 0 {
			rerankQuery = keywordQueries[0] // Use first keyword query as fallback
		}

		if rerankQuery != "" {
			logger.Debugf(ctx, "[Tool][KnowledgeSearch] Rerank query: %s", rerankQuery)
			rerankedResults, err := t.rerankResults(ctx, rerankQuery, filteredResults)
			if err != nil {
				logger.Warnf(ctx, "[Tool][KnowledgeSearch] Rerank failed, using original results: %v", err)
			} else {
				filteredResults = rerankedResults
				logger.Infof(ctx, "[Tool][KnowledgeSearch] Rerank completed successfully: %d results",
					len(filteredResults))
			}
		}
	}

	// Apply absolute minimum score filter to remove very low quality chunks
	logger.Debugf(ctx, "[Tool][KnowledgeSearch] Applying min_score filter (%.2f)...", minScore)
	filteredResults = t.filterByMinScore(filteredResults, minScore)
	logger.Infof(ctx, "[Tool][KnowledgeSearch] After min_score filter: %d results", len(filteredResults))

	logger.Debugf(ctx, "[Tool][KnowledgeSearch] Deduplicating results...")
	deduplicatedResults := t.deduplicateResults(filteredResults)
	logger.Infof(ctx, "[Tool][KnowledgeSearch] After deduplication: %d results (from %d)",
		len(deduplicatedResults), len(filteredResults))

	// Sort results by score (descending)
	logger.Debugf(ctx, "[Tool][KnowledgeSearch] Sorting results by score...")
	sort.Slice(deduplicatedResults, func(i, j int) bool {
		if deduplicatedResults[i].Score != deduplicatedResults[j].Score {
			return deduplicatedResults[i].Score > deduplicatedResults[j].Score
		}
		// If scores are equal, prefer vector matches
		if deduplicatedResults[i].QueryType != deduplicatedResults[j].QueryType {
			return deduplicatedResults[i].QueryType == "vector"
		}
		return deduplicatedResults[i].KnowledgeID < deduplicatedResults[j].KnowledgeID
	})

	// Log top results
	if len(deduplicatedResults) > 0 {
		logger.Infof(ctx, "[Tool][KnowledgeSearch] Top 5 results by score:")
		for i := 0; i < len(deduplicatedResults) && i < 5; i++ {
			r := deduplicatedResults[i]
			logger.Infof(ctx, "[Tool][KnowledgeSearch]   #%d: score=%.3f, type=%s, kb=%s, chunk_id=%s",
				i+1, r.Score, r.QueryType, r.KnowledgeID, r.ID)
		}
	}

	// Build output
	logger.Infof(ctx, "[Tool][KnowledgeSearch] Formatting output with %d final results", len(deduplicatedResults))
	result, err := t.formatOutput(deduplicatedResults, vectorQueries, keywordQueries,
		kbIDs, len(allResults), vectorThreshold, keywordThreshold, knowledgeIDsFilter, singleQuery)
	if err != nil {
		logger.Errorf(ctx, "[Tool][KnowledgeSearch] Failed to format output: %v", err)
		return result, err
	}

	logger.Infof(ctx, "[Tool][KnowledgeSearch] Execute completed successfully")
	return result, nil
}

// concurrentSearch executes vector and keyword searches concurrently
func (t *KnowledgeSearchTool) concurrentSearch(
	ctx context.Context,
	vectorQueries, keywordQueries []string,
	kbsToSearch []string,
	topK int,
	vectorThreshold, keywordThreshold float64,
) []*searchResultWithMeta {
	var wg sync.WaitGroup
	var mu sync.Mutex
	allResults := make([]*searchResultWithMeta, 0)

	// Launch vector searches
	if len(vectorQueries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := t.searchWithQueries(ctx, vectorQueries, kbsToSearch, topK,
				vectorThreshold, 1.0, "vector")
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}

	// Launch keyword searches
	if len(keywordQueries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := t.searchWithQueries(ctx, keywordQueries, kbsToSearch, topK,
				1.0, keywordThreshold, "keyword")
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return allResults
}

// searchWithQueries executes multiple queries concurrently
func (t *KnowledgeSearchTool) searchWithQueries(
	ctx context.Context,
	queries []string,
	kbsToSearch []string,
	topK int,
	vectorThreshold, keywordThreshold float64,
	queryType string,
) []*searchResultWithMeta {
	var wg sync.WaitGroup
	var mu sync.Mutex
	allResults := make([]*searchResultWithMeta, 0)

	for _, query := range queries {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()
			results := t.searchSingleQuery(ctx, q, kbsToSearch, topK,
				vectorThreshold, keywordThreshold, queryType)
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(query)
	}

	wg.Wait()
	return allResults
}

// searchSingleQuery searches a single query across multiple KBs concurrently
func (t *KnowledgeSearchTool) searchSingleQuery(
	ctx context.Context,
	query string,
	kbsToSearch []string,
	topK int,
	vectorThreshold, keywordThreshold float64,
	queryType string,
) []*searchResultWithMeta {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]*searchResultWithMeta, 0)

	searchParams := types.SearchParams{
		QueryText:        query,
		MatchCount:       topK,
		VectorThreshold:  vectorThreshold,
		KeywordThreshold: keywordThreshold,
	}

	for _, kbID := range kbsToSearch {
		wg.Add(1)
		go func(kb string) {
			defer wg.Done()

			kbResults, err := t.knowledgeService.HybridSearch(ctx, kb, searchParams)
			if err != nil {
				// Log error but continue with other KBs
				return
			}

			// Wrap results with metadata
			mu.Lock()
			for _, r := range kbResults {
				results = append(results, &searchResultWithMeta{
					SearchResult: r,
					SourceQuery:  query,
					QueryType:    queryType,
				})
			}
			mu.Unlock()
		}(kbID)
	}

	wg.Wait()
	return results
}

// filterByThreshold filters results based on match type and threshold
func (t *KnowledgeSearchTool) filterByThreshold(
	results []*searchResultWithMeta,
	vectorThreshold, keywordThreshold float64,
) []*searchResultWithMeta {
	filtered := make([]*searchResultWithMeta, 0)
	for _, r := range results {
		// Check if result meets threshold based on match type
		if r.MatchType == types.MatchTypeEmbedding && r.Score >= vectorThreshold {
			filtered = append(filtered, r)
		} else if r.MatchType == types.MatchTypeKeywords && r.Score >= keywordThreshold {
			filtered = append(filtered, r)
		} else {
			// For other match types (graph, nearby chunk, etc.), use the lower threshold
			minThreshold := vectorThreshold
			if keywordThreshold < minThreshold {
				minThreshold = keywordThreshold
			}
			if r.Score >= minThreshold {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// rerankResults applies reranking to search results using the configured rerank model
func (t *KnowledgeSearchTool) rerankResults(
	ctx context.Context,
	query string,
	results []*searchResultWithMeta,
) ([]*searchResultWithMeta, error) {
	// Prepare passages for reranking
	passages := make([]string, len(results))
	for i, result := range results {
		passages[i] = result.Content
	}

	// Call rerank model
	rerankResp, err := t.rerankModel.Rerank(ctx, query, passages)
	if err != nil {
		return nil, fmt.Errorf("rerank call failed: %w", err)
	}

	// Map reranked results back with new scores
	reranked := make([]*searchResultWithMeta, 0, len(rerankResp))
	for _, rr := range rerankResp {
		if rr.Index >= 0 && rr.Index < len(results) {
			// Create new result with reranked score
			newResult := *results[rr.Index]
			newResult.Score = rr.RelevanceScore
			reranked = append(reranked, &newResult)
		}
	}

	logger.Infof(ctx, "Reranked %d results from %d original results", len(reranked), len(results))
	return reranked, nil
}

// filterByMinScore filters results by absolute minimum score
func (t *KnowledgeSearchTool) filterByMinScore(
	results []*searchResultWithMeta,
	minScore float64,
) []*searchResultWithMeta {
	filtered := make([]*searchResultWithMeta, 0)
	for _, r := range results {
		if r.Score >= minScore {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// deduplicateResults removes duplicate chunks, keeping the highest score
func (t *KnowledgeSearchTool) deduplicateResults(results []*searchResultWithMeta) []*searchResultWithMeta {
	seen := make(map[string]*searchResultWithMeta)

	for _, r := range results {
		if existing, ok := seen[r.ID]; ok {
			// Keep the result with higher score
			if r.Score > existing.Score {
				seen[r.ID] = r
			}
		} else {
			seen[r.ID] = r
		}
	}

	deduplicated := make([]*searchResultWithMeta, 0, len(seen))
	for _, r := range seen {
		deduplicated = append(deduplicated, r)
	}

	return deduplicated
}

// formatOutput formats the search results for display
func (t *KnowledgeSearchTool) formatOutput(
	results []*searchResultWithMeta,
	vectorQueries, keywordQueries []string,
	kbsToSearch []string,
	totalBeforeFilter int,
	vectorThreshold, keywordThreshold float64,
	knowledgeIDsFilter map[string]bool,
	singleQuery string,
) (*types.ToolResult, error) {
	if len(results) == 0 {
		data := map[string]interface{}{
			"knowledge_base_ids": kbsToSearch,
			"results":            []interface{}{},
			"count":              0,
		}
		if len(knowledgeIDsFilter) > 0 {
			filterList := make([]string, 0, len(knowledgeIDsFilter))
			for id := range knowledgeIDsFilter {
				filterList = append(filterList, id)
			}
			data["knowledge_ids"] = filterList
		}
		if singleQuery != "" {
			data["query"] = singleQuery
		}
		return &types.ToolResult{
			Success: true,
			Output:  fmt.Sprintf("No relevant content found in %d knowledge base(s).", len(kbsToSearch)),
			Data:    data,
		}, nil
	}

	// Determine search mode
	searchMode := "Hybrid (Vector + Keyword)"
	if len(vectorQueries) > 0 && len(keywordQueries) == 0 {
		searchMode = "Vector"
	} else if len(vectorQueries) == 0 && len(keywordQueries) > 0 {
		searchMode = "Keyword"
	}

	// Build output header
	output := "=== Search Results ===\n"
	output += fmt.Sprintf("Knowledge Bases: %v\n", kbsToSearch)
	if len(knowledgeIDsFilter) > 0 {
		filterList := make([]string, 0, len(knowledgeIDsFilter))
		for id := range knowledgeIDsFilter {
			filterList = append(filterList, id)
		}
		output += fmt.Sprintf("Document Filter: %v\n", filterList)
	}
	output += fmt.Sprintf("Search Mode: %s\n", searchMode)

	if singleQuery != "" {
		output += fmt.Sprintf("Query: %s\n", singleQuery)
	} else {
		if len(vectorQueries) > 0 {
			output += fmt.Sprintf("Vector Queries: %v\n", vectorQueries)
			output += fmt.Sprintf("Vector Threshold: %.2f\n", vectorThreshold)
		}
		if len(keywordQueries) > 0 {
			output += fmt.Sprintf("Keyword Queries: %v\n", keywordQueries)
			output += fmt.Sprintf("Keyword Threshold: %.2f\n", keywordThreshold)
		}
	}

	output += fmt.Sprintf("Found %d relevant results (deduplicated)", len(results))
	if totalBeforeFilter > len(results) {
		output += fmt.Sprintf(" (filtered from %d)", totalBeforeFilter)
	}
	output += "\n\n"

	// Count results by KB
	kbCounts := make(map[string]int)
	for _, r := range results {
		kbCounts[r.KnowledgeID]++
	}

	output += "Knowledge Base Coverage:\n"
	for kbID, count := range kbCounts {
		output += fmt.Sprintf("  - %s: %d results\n", kbID, count)
	}
	output += "\n=== Detailed Results ===\n\n"

	// Format individual results
	formattedResults := make([]map[string]interface{}, 0, len(results))
	currentKB := ""

	for i, result := range results {
		// Group by knowledge base
		if result.KnowledgeID != currentKB {
			currentKB = result.KnowledgeID
			if i > 0 {
				output += "\n"
			}
			output += fmt.Sprintf("[Source Document: %s]\n", result.KnowledgeTitle)
		}

		relevanceLevel := GetRelevanceLevel(result.Score)
		output += fmt.Sprintf("\nResult #%d:\n", i+1)
		output += fmt.Sprintf("  Relevance: %.2f (%s)\n", result.Score, relevanceLevel)
		output += fmt.Sprintf("  Match Type: %s", FormatMatchType(result.MatchType))
		if result.SourceQuery != "" && result.SourceQuery != singleQuery {
			output += fmt.Sprintf(" (Query: \"%s\")", result.SourceQuery)
		}
		output += "\n"
		output += fmt.Sprintf("  Content: %s\n", result.Content)
		output += fmt.Sprintf("  [chunk_id: %s - full content included above]\n", result.ID)

		formattedResults = append(formattedResults, map[string]interface{}{
			"result_index":    i + 1,
			"chunk_id":        result.ID,
			"content":         result.Content,
			"score":           result.Score,
			"relevance_level": relevanceLevel,
			"knowledge_id":    result.KnowledgeID,
			"knowledge_title": result.KnowledgeTitle,
			"match_type":      result.MatchType,
			"source_query":    result.SourceQuery,
			"query_type":      result.QueryType,
		})
	}

	// Add usage guidance
	output += "\n\n=== Usage Guidelines ===\n"
	output += "- High relevance (>=0.8): directly usable for answering\n"
	output += "- Medium relevance (0.6-0.8): use as supplementary reference\n"
	output += "- Low relevance (<0.6): use with caution, may not be accurate\n"
	if totalBeforeFilter > len(results) {
		output += "- Results below threshold have been automatically filtered\n"
	}
	output += "- Full content is already included in search results above\n"
	output += "- Results are deduplicated across knowledge bases and sorted by relevance\n"
	output += "- Use get_related_chunks to expand context if needed\n"

	data := map[string]interface{}{
		"knowledge_base_ids": kbsToSearch,
		"results":            formattedResults,
		"count":              len(results),
		"kb_counts":          kbCounts,
		"search_mode":        searchMode,
		"display_type":       "search_results",
	}
	if len(knowledgeIDsFilter) > 0 {
		filterList := make([]string, 0, len(knowledgeIDsFilter))
		for id := range knowledgeIDsFilter {
			filterList = append(filterList, id)
		}
		data["knowledge_ids"] = filterList
	}
	if singleQuery != "" {
		data["query"] = singleQuery
	}
	if len(vectorQueries) > 0 {
		data["vector_queries"] = vectorQueries
	}
	if len(keywordQueries) > 0 {
		data["keyword_queries"] = keywordQueries
	}
	if totalBeforeFilter > len(results) {
		data["total_before_filter"] = totalBeforeFilter
		data["total_after_filter"] = len(results)
	}

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data:    data,
	}, nil
}
