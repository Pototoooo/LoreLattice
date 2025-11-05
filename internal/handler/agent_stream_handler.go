package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// AgentStreamHandler handles agent events for SSE streaming
// It uses a dedicated EventBus per request to avoid SessionID filtering
type AgentStreamHandler struct {
	ctx                context.Context
	ginContext         *gin.Context
	sessionID          string
	assistantMessageID string
	requestID          string
	assistantMessage   *types.Message
	streamManager      interfaces.StreamManager

	eventBus *event.EventBus

	// State tracking
	knowledgeRefs      []*types.SearchResult
	finalAnswer        string
	accumulatedContent map[string]string    // Accumulate content per event ID (thought, answer, reflection)
	eventStartTimes    map[string]time.Time // Track start time for duration calculation
	mu                 sync.Mutex
}

// NewAgentStreamHandler creates a new handler for agent SSE streaming
func NewAgentStreamHandler(
	ctx context.Context,
	c *gin.Context,
	sessionID, assistantMessageID, requestID string,
	assistantMessage *types.Message,
	streamManager interfaces.StreamManager,
	eventBus *event.EventBus,
) *AgentStreamHandler {
	return &AgentStreamHandler{
		ctx:                ctx,
		ginContext:         c,
		sessionID:          sessionID,
		assistantMessageID: assistantMessageID,
		requestID:          requestID,
		assistantMessage:   assistantMessage,
		streamManager:      streamManager,
		eventBus:           eventBus,
		knowledgeRefs:      make([]*types.SearchResult, 0),
		accumulatedContent: make(map[string]string),
		eventStartTimes:    make(map[string]time.Time),
	}
}

// Subscribe subscribes to all agent streaming events on the dedicated EventBus
// No SessionID filtering needed since we have a dedicated EventBus per request
func (h *AgentStreamHandler) Subscribe() {
	// Subscribe to all agent streaming events on the dedicated EventBus
	h.eventBus.On(event.EventAgentThought, h.handleThought)
	h.eventBus.On(event.EventAgentToolCall, h.handleToolCall)
	h.eventBus.On(event.EventAgentToolResult, h.handleToolResult)
	h.eventBus.On(event.EventAgentReferences, h.handleReferences)
	h.eventBus.On(event.EventAgentFinalAnswer, h.handleFinalAnswer)
	h.eventBus.On(event.EventAgentReflection, h.handleReflection)
	h.eventBus.On(event.EventError, h.handleError)
	h.eventBus.On(event.EventAgentComplete, h.handleComplete)
}

// handleThought handles agent thought events
func (h *AgentStreamHandler) handleThought(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentThoughtData)
	if !ok {
		return nil
	}

	h.mu.Lock()

	// Track start time on first chunk
	if _, exists := h.eventStartTimes[evt.ID]; !exists {
		h.eventStartTimes[evt.ID] = time.Now()
	}

	// Calculate duration if done
	var metadata map[string]interface{}
	if data.Done {
		startTime := h.eventStartTimes[evt.ID]
		duration := time.Since(startTime)
		metadata = map[string]interface{}{
			"event_id":     evt.ID,
			"duration_ms":  duration.Milliseconds(),
			"completed_at": time.Now().Unix(),
		}
		delete(h.eventStartTimes, evt.ID)
	} else {
		metadata = map[string]interface{}{
			"event_id": evt.ID,
		}
	}

	h.mu.Unlock()

	// Send SSE response (real-time incremental)
	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: types.ResponseTypeThinking,
		Content:      data.Content,
		Done:         data.Done,
		Data:         metadata,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Accumulate content and replace event in stream (so refresh can see progress)
	h.mu.Lock()
	h.accumulateAndReplaceEventWithData(evt.ID, types.ResponseTypeThinking, data.Content, data.Done, metadata)
	h.mu.Unlock()

	return nil
}

// handleToolCall handles tool call events
func (h *AgentStreamHandler) handleToolCall(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentToolCallData)
	if !ok {
		return nil
	}

	metadata := map[string]interface{}{
		"tool_name": data.ToolName,
		"arguments": data.Arguments,
	}

	// Send SSE response
	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: types.ResponseTypeToolCall,
		Content:      fmt.Sprintf("Calling tool: %s", data.ToolName),
		Done:         false,
		Data:         metadata,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Push event to stream for replay on refresh
	if err := h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
		ID:        evt.ID,
		Type:      types.ResponseTypeToolCall,
		Content:   fmt.Sprintf("Calling tool: %s", data.ToolName),
		Done:      false,
		Timestamp: time.Now(),
		Data:      metadata,
	}); err != nil {
		logger.GetLogger(h.ctx).Error("Push tool call event to stream failed", "error", err)
	}

	return nil
}

// handleToolResult handles tool result events
func (h *AgentStreamHandler) handleToolResult(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentToolResultData)
	if !ok {
		return nil
	}

	// Send SSE response (both success and failure)
	responseType := types.ResponseTypeToolResult
	content := data.Output
	if !data.Success {
		responseType = types.ResponseTypeError
		if data.Error != "" {
			content = data.Error
		}
	}

	// Build metadata including tool result data for rich frontend rendering
	metadata := map[string]interface{}{
		"tool_name": data.ToolName,
		"success":   data.Success,
		"output":    data.Output,
		"error":     data.Error,
		"duration":  data.Duration,
	}

	// Merge tool result data (contains display_type, formatted results, etc.)
	if data.Data != nil {
		for k, v := range data.Data {
			metadata[k] = v
		}
	}

	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: responseType,
		Content:      content,
		Done:         false,
		Data:         metadata,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Push event to stream for replay on refresh
	if err := h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
		ID:        evt.ID,
		Type:      responseType,
		Content:   content,
		Done:      false,
		Timestamp: time.Now(),
		Data:      metadata,
	}); err != nil {
		logger.GetLogger(h.ctx).Error("Push tool result event to stream failed", "error", err)
	}

	return nil
}

// handleReferences handles knowledge references events
func (h *AgentStreamHandler) handleReferences(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentReferencesData)
	if !ok {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Extract knowledge references
	// Try to cast directly to []*types.SearchResult first
	if searchResults, ok := data.References.([]*types.SearchResult); ok {
		h.knowledgeRefs = append(h.knowledgeRefs, searchResults...)
	} else if refs, ok := data.References.([]interface{}); ok {
		// Fallback: convert from []interface{}
		for _, ref := range refs {
			if sr, ok := ref.(*types.SearchResult); ok {
				h.knowledgeRefs = append(h.knowledgeRefs, sr)
			} else if refMap, ok := ref.(map[string]interface{}); ok {
				// Parse from map if needed
				searchResult := &types.SearchResult{
					ID:             getString(refMap, "id"),
					Content:        getString(refMap, "content"),
					Score:          getFloat64(refMap, "score"),
					KnowledgeID:    getString(refMap, "knowledge_id"),
					KnowledgeTitle: getString(refMap, "knowledge_title"),
					ChunkIndex:     int(getFloat64(refMap, "chunk_index")),
				}

				if meta, ok := refMap["metadata"].(map[string]interface{}); ok {
					metadata := make(map[string]string)
					for k, v := range meta {
						if strVal, ok := v.(string); ok {
							metadata[k] = strVal
						}
					}
					searchResult.Metadata = metadata
				}

				h.knowledgeRefs = append(h.knowledgeRefs, searchResult)
			}
		}
	}

	// Update assistant message references
	h.assistantMessage.KnowledgeReferences = h.knowledgeRefs

	// Send SSE response
	response := types.StreamResponse{
		ID:                  h.requestID,
		ResponseType:        types.ResponseTypeReferences,
		KnowledgeReferences: h.knowledgeRefs,
		Done:                false,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Update stream references
	if err := h.streamManager.UpdateReferences(
		h.ctx, h.sessionID, h.assistantMessageID, h.knowledgeRefs,
	); err != nil {
		logger.GetLogger(h.ctx).Error("Update stream references failed", "error", err)
	}

	// Push event to stream for replay on refresh
	if err := h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
		ID:        evt.ID,
		Type:      types.ResponseTypeReferences,
		Content:   "",
		Done:      false,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"references": h.knowledgeRefs,
		},
	}); err != nil {
		logger.GetLogger(h.ctx).Error("Push references event to stream failed", "error", err)
	}

	return nil
}

// handleFinalAnswer handles final answer events
func (h *AgentStreamHandler) handleFinalAnswer(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentFinalAnswerData)
	if !ok {
		return nil
	}

	h.mu.Lock()

	// Accumulate final answer (this persists and won't be reset)
	h.finalAnswer += data.Content
	h.assistantMessage.Content = h.finalAnswer

	// Send SSE response (real-time incremental)
	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: types.ResponseTypeAnswer,
		Content:      data.Content,
		Done:         data.Done,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Replace answer event in stream (so refresh can see progress)
	h.accumulateAndReplaceEvent(evt.ID, types.ResponseTypeAnswer, data.Content, data.Done)
	h.mu.Unlock()

	return nil
}

// handleReflection handles agent reflection events
func (h *AgentStreamHandler) handleReflection(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentReflectionData)
	if !ok {
		return nil
	}

	// Send reflection as SSE (real-time incremental)
	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: "reflection", // Special type for reflection
		Content:      data.Content,
		Done:         data.Done,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	// Accumulate reflection per tool call and replace event in stream
	h.mu.Lock()
	h.accumulateAndReplaceEvent(evt.ID, types.ResponseTypeReflection, data.Content, data.Done)
	h.mu.Unlock()

	return nil
}

// handleError handles error events
func (h *AgentStreamHandler) handleError(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.ErrorData)
	if !ok {
		return nil
	}

	response := types.StreamResponse{
		ID:           h.requestID,
		ResponseType: types.ResponseTypeError,
		Content:      data.Error,
		Done:         true,
	}

	h.ginContext.SSEvent("message", response)
	h.ginContext.Writer.Flush()

	return nil
}

// handleComplete handles agent complete events
func (h *AgentStreamHandler) handleComplete(ctx context.Context, evt event.Event) error {
	data, ok := evt.Data.(event.AgentCompleteData)
	if !ok {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Update assistant message with final data
	if data.MessageID == h.assistantMessageID {
		h.assistantMessage.Content = data.FinalAnswer
		h.assistantMessage.IsCompleted = true

		// Update knowledge references if provided
		if len(data.KnowledgeRefs) > 0 {
			knowledgeRefs := make([]*types.SearchResult, 0, len(data.KnowledgeRefs))
			for _, ref := range data.KnowledgeRefs {
				if sr, ok := ref.(*types.SearchResult); ok {
					knowledgeRefs = append(knowledgeRefs, sr)
				}
			}
			h.assistantMessage.KnowledgeReferences = knowledgeRefs
		}

		// Update agent steps if provided
		if data.AgentSteps != nil {
			if steps, ok := data.AgentSteps.([]types.AgentStep); ok {
				h.assistantMessage.AgentSteps = steps
			}
		}
	}

	// Complete stream
	if err := h.streamManager.CompleteStream(h.ctx, h.sessionID, h.assistantMessageID); err != nil {
		logger.GetLogger(h.ctx).Error("Complete stream failed", "error", err)
	}

	return nil
}

// accumulateAndReplaceEvent accumulates content per event ID and replaces the event in stream
// This is a common pattern for thought, answer, and reflection events
func (h *AgentStreamHandler) accumulateAndReplaceEvent(
	evtID string,
	evtType types.ResponseType,
	content string,
	done bool,
) {
	h.accumulateAndReplaceEventWithData(evtID, evtType, content, done, nil)
}

// accumulateAndReplaceEventWithData accumulates content and replaces event with metadata
func (h *AgentStreamHandler) accumulateAndReplaceEventWithData(
	evtID string,
	evtType types.ResponseType,
	content string,
	done bool,
	data map[string]interface{},
) {
	h.accumulatedContent[evtID] += content
	if err := h.streamManager.ReplaceEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
		ID:        evtID,
		Type:      evtType,
		Content:   h.accumulatedContent[evtID],
		Done:      done,
		Timestamp: time.Now(),
		Data:      data,
	}); err != nil {
		logger.GetLogger(h.ctx).Error(fmt.Sprintf("Replace %s event in stream failed", evtType), "error", err)
	}
	// Clean up when done
	if done {
		delete(h.accumulatedContent, evtID)
	}
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return 0.0
}
