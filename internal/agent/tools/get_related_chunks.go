package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// GetRelatedChunksTool retrieves chunks related to a given chunk
type GetRelatedChunksTool struct {
	BaseTool
	chunkService         interfaces.ChunkService
	knowledgeBaseService interfaces.KnowledgeBaseService
}

// NewGetRelatedChunksTool creates a new get related chunks tool
func NewGetRelatedChunksTool(chunkService interfaces.ChunkService, knowledgeBaseService interfaces.KnowledgeBaseService) *GetRelatedChunksTool {
	description := `Retrieve chunks related to specified reference chunks. Supports sequential (adjacent) and semantic (similar) relation types.

## When to Use

Use this tool when:
- Search results need additional context for full understanding
- You need to see content before/after a specific chunk
- Looking for semantically similar content across the document
- Understanding the complete narrative flow of a topic

Do not use when:
- Search results already provide sufficient complete content
- Only need a single specific chunk without context

## Parameters

chunk_ids (required): Array of reference chunk IDs (1-10)
- Obtained from search results
- Supports concurrent batch processing
- Example: ["chunk_abc", "chunk_def"]

relation_type (optional): Type of relation
- "sequential" (default): Get adjacent chunks before and after
- "semantic": Get semantically similar chunks regardless of position

limit (optional): Number of related chunks to return per reference chunk
- Default: 5
- Range: 1-10
- Sequential: retrieves limit/2 chunks before and after
- Semantic: retrieves top limit most similar chunks

## Relation Types

Sequential:
- Retrieves adjacent chunks in document order
- Useful for understanding complete narrative flow
- Ideal for scenarios requiring continuous reading
- Example: viewing complete configuration steps

Semantic:
- Finds content-similar chunks regardless of position
- Discovers related discussions throughout document
- Ideal for topic expansion and cross-referencing
- Example: finding all mentions of a specific concept

## Usage Patterns

1. Context expansion: knowledge_search -> get_related_chunks(sequential)
2. Topic exploration: knowledge_search -> get_related_chunks(semantic)  
3. Deep research: knowledge_search -> get_related_chunks(both sequential and semantic)

## Notes

- Results are automatically deduplicated
- Source chunks are excluded from results
- Sequential results sorted by chunk_index
- Semantic results sorted by similarity score
- Limit value of 5 typically provides sufficient context without information overload`

	return &GetRelatedChunksTool{
		BaseTool:             NewBaseTool("get_related_chunks", description),
		chunkService:         chunkService,
		knowledgeBaseService: knowledgeBaseService,
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *GetRelatedChunksTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"chunk_ids": map[string]interface{}{
				"type":        "array",
				"description": "Array of reference chunk IDs",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 10,
			},
			"relation_type": map[string]interface{}{
				"type":        "string",
				"description": "Type: sequential (default) or semantic",
				"enum":        []string{"sequential", "semantic"},
				"default":     "sequential",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Number of related chunks per input chunk (default: 5)",
				"default":     5,
				"minimum":     1,
				"maximum":     10,
			},
		},
		"required": []string{"chunk_ids"},
	}
}

// Execute executes the get related chunks tool with concurrent processing
func (t *GetRelatedChunksTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	// Extract chunk_ids array
	chunkIDsRaw, ok := args["chunk_ids"].([]interface{})
	if !ok || len(chunkIDsRaw) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "chunk_ids is required and must be a non-empty array",
		}, fmt.Errorf("chunk_ids is required")
	}

	// Convert to string slice
	var chunkIDs []string
	for _, id := range chunkIDsRaw {
		if idStr, ok := id.(string); ok && idStr != "" {
			chunkIDs = append(chunkIDs, idStr)
		}
	}

	if len(chunkIDs) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "chunk_ids must contain at least one valid chunk ID",
		}, fmt.Errorf("no valid chunk IDs provided")
	}

	relationType := "sequential"
	if rt, ok := args["relation_type"].(string); ok {
		relationType = rt
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 10 {
		limit = 10
	}

	// Concurrently get related chunks for each chunk ID
	type relatedResult struct {
		sourceChunk   *types.Chunk
		relatedChunks []*types.Chunk
		err           error
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]*relatedResult)

	for _, chunkID := range chunkIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Get the original chunk first
			chunk, err := t.chunkService.GetChunkByID(ctx, id)
			if err != nil || chunk == nil {
				mu.Lock()
				results[id] = &relatedResult{err: fmt.Errorf("failed to retrieve chunk: %v", err)}
				mu.Unlock()
				return
			}

			var relatedChunks []*types.Chunk

			if relationType == "sequential" {
				relatedChunks, err = t.getSequentialRelatedChunks(ctx, chunk, limit)
			} else if relationType == "semantic" {
				relatedChunks, err = t.getSemanticRelatedChunks(ctx, chunk, limit)
			}

			mu.Lock()
			results[id] = &relatedResult{
				sourceChunk:   chunk,
				relatedChunks: relatedChunks,
				err:           err,
			}
			mu.Unlock()
		}(chunkID)
	}

	wg.Wait()

	// Collect and deduplicate all related chunks
	seenChunks := make(map[string]*types.Chunk)
	sourceChunkIDs := make(map[string]bool)
	var errors []string

	// Mark source chunks to exclude them from results
	for _, chunkID := range chunkIDs {
		sourceChunkIDs[chunkID] = true
	}

	for _, chunkID := range chunkIDs {
		result := results[chunkID]
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("chunk %s: %v", chunkID, result.err))
			continue
		}

		for _, chunk := range result.relatedChunks {
			// Exclude source chunks and avoid duplicates
			if !sourceChunkIDs[chunk.ID] {
				if _, seen := seenChunks[chunk.ID]; !seen {
					seenChunks[chunk.ID] = chunk
				}
			}
		}
	}

	// Convert map to slice and sort
	allRelatedChunks := make([]*types.Chunk, 0, len(seenChunks))
	for _, chunk := range seenChunks {
		allRelatedChunks = append(allRelatedChunks, chunk)
	}

	// Sort chunks
	if relationType == "sequential" {
		// Sort by knowledge_id and chunk_index for sequential
		sortChunksByPosition(allRelatedChunks)
	}
	// For semantic, keep the order from search results (already sorted by relevance)

	if len(allRelatedChunks) == 0 {
		return &types.ToolResult{
			Success: true,
			Output:  "No related chunks found. Possible reasons:\n- Chunk is the only chunk in document\n- Semantic similarity threshold not met\n- Invalid chunk_id provided",
			Data: map[string]interface{}{
				"chunk_ids":     chunkIDs,
				"relation_type": relationType,
				"count":         0,
				"chunks":        []interface{}{},
				"errors":        errors,
			},
		}, nil
	}

	// Format output
	return t.formatOutput(chunkIDs, relationType, allRelatedChunks, errors)
}

// getSequentialRelatedChunks gets chunks before and after the reference chunk
func (t *GetRelatedChunksTool) getSequentialRelatedChunks(ctx context.Context, chunk *types.Chunk, limit int) ([]*types.Chunk, error) {
	// Get all chunks from the same knowledge
	allChunks, err := t.chunkService.ListChunksByKnowledgeID(ctx, chunk.KnowledgeID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve document chunks: %v", err)
	}

	relatedChunks := make([]*types.Chunk, 0)

	// Bidirectional window: get chunks before and after
	halfLimit := limit / 2
	if halfLimit < 1 {
		halfLimit = 1
	}

	minIndex := chunk.ChunkIndex - halfLimit
	maxIndex := chunk.ChunkIndex + halfLimit

	for _, c := range allChunks {
		// Within range and not the source chunk itself
		if c.ChunkIndex >= minIndex && c.ChunkIndex <= maxIndex && c.ID != chunk.ID {
			relatedChunks = append(relatedChunks, c)
		}
	}

	return relatedChunks, nil
}

// getSemanticRelatedChunks gets semantically similar chunks using hybrid search
func (t *GetRelatedChunksTool) getSemanticRelatedChunks(ctx context.Context, chunk *types.Chunk, limit int) ([]*types.Chunk, error) {
	// Use chunk content as query for semantic search
	searchParams := types.SearchParams{
		QueryText:  chunk.Content,
		MatchCount: limit + 5, // Get extra results for filtering
	}

	// Search in the knowledge base that contains this chunk
	searchResults, err := t.knowledgeBaseService.HybridSearch(ctx, chunk.KnowledgeBaseID, searchParams)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %v", err)
	}

	// Convert search results to chunks, excluding the source chunk
	relatedChunks := make([]*types.Chunk, 0, limit)
	for _, result := range searchResults {
		if result.ID == chunk.ID {
			continue // Skip the source chunk itself
		}

		// Convert SearchResult to Chunk
		relatedChunk := &types.Chunk{
			ID:              result.ID,
			KnowledgeID:     result.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			Content:         result.Content,
			ChunkIndex:      result.ChunkIndex,
		}

		relatedChunks = append(relatedChunks, relatedChunk)

		if len(relatedChunks) >= limit {
			break
		}
	}

	return relatedChunks, nil
}

// sortChunksByPosition sorts chunks by knowledge_id and chunk_index
func sortChunksByPosition(chunks []*types.Chunk) {
	// Simple bubble sort for small arrays
	n := len(chunks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			// First sort by knowledge_id, then by chunk_index
			if chunks[j].KnowledgeID > chunks[j+1].KnowledgeID ||
				(chunks[j].KnowledgeID == chunks[j+1].KnowledgeID &&
					chunks[j].ChunkIndex > chunks[j+1].ChunkIndex) {
				chunks[j], chunks[j+1] = chunks[j+1], chunks[j]
			}
		}
	}
}

// formatOutput formats the tool output
func (t *GetRelatedChunksTool) formatOutput(
	chunkIDs []string,
	relationType string,
	chunks []*types.Chunk,
	errors []string,
) (*types.ToolResult, error) {
	relationTypeLabel := map[string]string{
		"sequential": "Sequential (Adjacent)",
		"semantic":   "Semantic (Similar Content)",
	}

	output := "=== Related Chunks ===\n\n"
	output += fmt.Sprintf("Reference chunks: %d\n", len(chunkIDs))
	output += fmt.Sprintf("Relation type: %s\n", relationTypeLabel[relationType])
	output += fmt.Sprintf("Found %d related chunks (deduplicated)\n\n", len(chunks))

	if len(errors) > 0 {
		output += "=== Partial Failures ===\n"
		for _, errMsg := range errors {
			output += fmt.Sprintf("  - %s\n", errMsg)
		}
		output += "\n"
	}

	output += "=== Content ===\n\n"

	formattedChunks := make([]map[string]interface{}, 0, len(chunks))
	currentKnowledge := ""

	for i, c := range chunks {
		// Group by knowledge document
		if c.KnowledgeID != currentKnowledge {
			currentKnowledge = c.KnowledgeID
			if i > 0 {
				output += "\n"
			}
			output += fmt.Sprintf("[Document: %s]\n\n", c.KnowledgeID)
		}

		output += fmt.Sprintf("Chunk #%d (Position: %d):\n", i+1, c.ChunkIndex+1)
		output += fmt.Sprintf("  chunk_id: %s\n", c.ID)
		output += fmt.Sprintf("  content: %s\n\n", c.Content)

		formattedChunks = append(formattedChunks, map[string]interface{}{
			"index":        i + 1,
			"chunk_id":     c.ID,
			"chunk_index":  c.ChunkIndex,
			"content":      c.Content,
			"knowledge_id": c.KnowledgeID,
		})
	}

	output += "=== Notes ===\n"
	if relationType == "sequential" {
		output += "- Adjacent chunks in document order\n"
		output += "- Useful for understanding complete narrative flow\n"
		output += "- Sorted by position\n"
	} else {
		output += "- Semantically similar chunks sorted by relevance\n"
		output += "- Useful for discovering related discussions\n"
		output += "- Ideal for topic expansion and cross-referencing\n"
	}
	output += "- Source chunks excluded\n"
	output += "- Results deduplicated\n"

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"chunk_ids":     chunkIDs,
			"relation_type": relationType,
			"count":         len(chunks),
			"chunks":        formattedChunks,
			"errors":        errors,
			"display_type":  "related_chunks",
		},
	}, nil
}
