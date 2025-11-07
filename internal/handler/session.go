package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SessionHandler handles all HTTP requests related to conversation sessions
type SessionHandler struct {
	messageService       interfaces.MessageService // Service for managing messages
	sessionService       interfaces.SessionService // Service for managing sessions
	streamManager        interfaces.StreamManager  // Manager for handling streaming responses
	config               *config.Config            // Application configuration
	knowledgebaseService interfaces.KnowledgeBaseService
}

// NewSessionHandler creates a new instance of SessionHandler with all necessary dependencies
func NewSessionHandler(
	sessionService interfaces.SessionService,
	messageService interfaces.MessageService,
	streamManager interfaces.StreamManager,
	config *config.Config,
	knowledgebaseService interfaces.KnowledgeBaseService,
) *SessionHandler {
	handler := &SessionHandler{
		sessionService:       sessionService,
		messageService:       messageService,
		streamManager:        streamManager,
		config:               config,
		knowledgebaseService: knowledgebaseService,
	}
	return handler
}

// SessionStrategy defines the configuration for a conversation session strategy
type SessionStrategy struct {
	// Maximum number of conversation rounds to maintain
	MaxRounds int `json:"max_rounds"`
	// Whether to enable query rewrite for multi-round conversations
	EnableRewrite bool `json:"enable_rewrite"`
	// Strategy to use when no relevant knowledge is found
	FallbackStrategy types.FallbackStrategy `json:"fallback_strategy"`
	// Fixed response content for fallback
	FallbackResponse string `json:"fallback_response"`
	// Number of top results to retrieve from vector search
	EmbeddingTopK int `json:"embedding_top_k"`
	// Threshold for keyword-based retrieval
	KeywordThreshold float64 `json:"keyword_threshold"`
	// Threshold for vector-based retrieval
	VectorThreshold float64 `json:"vector_threshold"`
	// ID of the model used for reranking results
	RerankModelID string `json:"rerank_model_id"`
	// Number of top results after reranking
	RerankTopK int `json:"rerank_top_k"`
	// Threshold for reranking results
	RerankThreshold float64 `json:"rerank_threshold"`
	// ID of the model used for summarization
	SummaryModelID string `json:"summary_model_id"`
	// Parameters for the summary model
	SummaryParameters *types.SummaryConfig `json:"summary_parameters" gorm:"type:json"`
	// Prefix for responses when no match is found
	NoMatchPrefix string `json:"no_match_prefix"`
}

// CreateSessionRequest represents a request to create a new session
// Sessions are now knowledge-base-independent and serve as conversation containers.
// Knowledge bases can be specified dynamically in each query request (AgentQA/KnowledgeQA).
type CreateSessionRequest struct {
	// ID of the associated knowledge base (optional, can be set/changed during queries)
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// Session strategy configuration
	SessionStrategy *SessionStrategy `json:"session_strategy"`
	// Agent configuration (optional, session-level config only: enabled and knowledge_bases)
	AgentConfig *types.SessionAgentConfig `json:"agent_config"`
}

// CreateSession handles the creation of a new conversation session
func (h *SessionHandler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	// Parse and validate the request body
	var request CreateSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to validate session creation parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Get tenant ID from context
	tenantID, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Failed to get tenant ID")
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	// Validate session creation request
	// Sessions are now knowledge-base-independent:
	// - KnowledgeBaseID is optional during session creation
	// - Knowledge base can be specified in each query request (AgentQA/KnowledgeQA)
	// - Agent mode can access multiple knowledge bases via AgentConfig.KnowledgeBases
	// - Knowledge base can be switched during conversation
	isAgentMode := request.AgentConfig != nil && request.AgentConfig.Enabled
	hasAgentKnowledgeBases := request.AgentConfig != nil && len(request.AgentConfig.KnowledgeBases) > 0

	logger.Infof(
		ctx,
		"Processing session creation request, tenant ID: %d, knowledge base ID: %s, agent mode: %v, agent KBs: %v",
		tenantID.(uint),
		request.KnowledgeBaseID,
		isAgentMode,
		hasAgentKnowledgeBases,
	)

	// Create session object with base properties
	createdSession := &types.Session{
		TenantID:        tenantID.(uint),
		KnowledgeBaseID: request.KnowledgeBaseID,
		AgentConfig:     request.AgentConfig, // Set agent config if provided
	}

	// If summary model parameters are empty, set defaults
	if request.SessionStrategy != nil {
		createdSession.RerankModelID = request.SessionStrategy.RerankModelID
		createdSession.SummaryModelID = request.SessionStrategy.SummaryModelID
		createdSession.MaxRounds = request.SessionStrategy.MaxRounds
		createdSession.EnableRewrite = request.SessionStrategy.EnableRewrite
		createdSession.FallbackStrategy = request.SessionStrategy.FallbackStrategy
		createdSession.FallbackResponse = request.SessionStrategy.FallbackResponse
		createdSession.EmbeddingTopK = request.SessionStrategy.EmbeddingTopK
		createdSession.KeywordThreshold = request.SessionStrategy.KeywordThreshold
		createdSession.VectorThreshold = request.SessionStrategy.VectorThreshold
		createdSession.RerankTopK = request.SessionStrategy.RerankTopK
		createdSession.RerankThreshold = request.SessionStrategy.RerankThreshold
		if request.SessionStrategy.SummaryParameters != nil {
			createdSession.SummaryParameters = request.SessionStrategy.SummaryParameters
		} else {
			createdSession.SummaryParameters = &types.SummaryConfig{
				MaxTokens:           h.config.Conversation.Summary.MaxTokens,
				TopP:                h.config.Conversation.Summary.TopP,
				TopK:                h.config.Conversation.Summary.TopK,
				FrequencyPenalty:    h.config.Conversation.Summary.FrequencyPenalty,
				PresencePenalty:     h.config.Conversation.Summary.PresencePenalty,
				RepeatPenalty:       h.config.Conversation.Summary.RepeatPenalty,
				NoMatchPrefix:       h.config.Conversation.Summary.NoMatchPrefix,
				Temperature:         h.config.Conversation.Summary.Temperature,
				Seed:                h.config.Conversation.Summary.Seed,
				MaxCompletionTokens: h.config.Conversation.Summary.MaxCompletionTokens,
			}
		}
		if createdSession.SummaryParameters.Prompt == "" {
			createdSession.SummaryParameters.Prompt = h.config.Conversation.Summary.Prompt
		}
		if createdSession.SummaryParameters.ContextTemplate == "" {
			createdSession.SummaryParameters.ContextTemplate = h.config.Conversation.Summary.ContextTemplate
		}
		if createdSession.SummaryParameters.NoMatchPrefix == "" {
			createdSession.SummaryParameters.NoMatchPrefix = h.config.Conversation.Summary.NoMatchPrefix
		}

		logger.Debug(ctx, "Custom session strategy set")
	} else {
		// Use default configuration from global config
		createdSession.MaxRounds = h.config.Conversation.MaxRounds
		createdSession.EnableRewrite = h.config.Conversation.EnableRewrite
		createdSession.FallbackStrategy = types.FallbackStrategy(h.config.Conversation.FallbackStrategy)
		createdSession.FallbackResponse = h.config.Conversation.FallbackResponse
		createdSession.EmbeddingTopK = h.config.Conversation.EmbeddingTopK
		createdSession.KeywordThreshold = h.config.Conversation.KeywordThreshold
		createdSession.VectorThreshold = h.config.Conversation.VectorThreshold
		createdSession.RerankThreshold = h.config.Conversation.RerankThreshold
		createdSession.RerankTopK = h.config.Conversation.RerankTopK
		createdSession.SummaryParameters = &types.SummaryConfig{
			MaxTokens:           h.config.Conversation.Summary.MaxTokens,
			TopP:                h.config.Conversation.Summary.TopP,
			TopK:                h.config.Conversation.Summary.TopK,
			FrequencyPenalty:    h.config.Conversation.Summary.FrequencyPenalty,
			PresencePenalty:     h.config.Conversation.Summary.PresencePenalty,
			RepeatPenalty:       h.config.Conversation.Summary.RepeatPenalty,
			Prompt:              h.config.Conversation.Summary.Prompt,
			ContextTemplate:     h.config.Conversation.Summary.ContextTemplate,
			NoMatchPrefix:       h.config.Conversation.Summary.NoMatchPrefix,
			Temperature:         h.config.Conversation.Summary.Temperature,
			Seed:                h.config.Conversation.Summary.Seed,
			MaxCompletionTokens: h.config.Conversation.Summary.MaxCompletionTokens,
		}

		logger.Debug(ctx, "Using default session strategy")
	}

	// Fetch knowledge base if KnowledgeBaseID is provided to inherit its model configurations
	// If no KB is provided, models will be determined at query time or use tenant/system defaults
	if request.KnowledgeBaseID != "" {
		kb, err := h.knowledgebaseService.GetKnowledgeBaseByID(ctx, request.KnowledgeBaseID)
		if err != nil {
			logger.Error(ctx, "Failed to get knowledge base", err)
			c.Error(errors.NewInternalServerError(err.Error()))
			return
		}

		// Use knowledge base's models if session doesn't specify them
		if createdSession.RerankModelID == "" {
			createdSession.RerankModelID = kb.RerankModelID
		}
		if createdSession.SummaryModelID == "" {
			createdSession.SummaryModelID = kb.SummaryModelID
		}

		logger.Debugf(ctx, "Knowledge base fetched: %s, rerank model: %s, summary model: %s",
			kb.ID, kb.RerankModelID, kb.SummaryModelID)
	} else {
		logger.Debug(ctx, "No knowledge base ID provided, models will use session strategy or be determined at query time")
	}

	// Call service to create session
	logger.Infof(ctx, "Calling session service to create session")
	createdSession, err := h.sessionService.CreateSession(ctx, createdSession)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return created session
	logger.Infof(ctx, "Session created successfully, ID: %s", createdSession.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdSession,
	})
}

// GetSession retrieves a session by its ID
func (h *SessionHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving session")

	// Get session ID from URL parameter
	id := c.Param("id")
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Call service to get session details
	logger.Infof(ctx, "Retrieving session, ID: %s", id)
	session, err := h.sessionService.GetSession(ctx, id)
	if err != nil {
		if err == errors.ErrSessionNotFound {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return session data
	logger.Infof(ctx, "Session retrieved successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    session,
	})
}

// GetSessionsByTenant retrieves all sessions for the current tenant with pagination
func (h *SessionHandler) GetSessionsByTenant(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving all sessions for tenant")

	// Parse pagination parameters from query
	var pagination types.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		logger.Error(ctx, "Failed to parse pagination parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Debugf(ctx, "Using pagination parameters: page=%d, page_size=%d", pagination.Page, pagination.PageSize)

	// Use paginated query to get sessions
	result, err := h.sessionService.GetPagedSessionsByTenant(ctx, &pagination)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return sessions with pagination data
	logger.Infof(ctx, "Successfully retrieved tenant sessions, total: %d", result.Total)
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result.Data,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

// UpdateSession updates an existing session's properties
func (h *SessionHandler) UpdateSession(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating session")

	// Get session ID from URL parameter
	id := c.Param("id")
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Verify tenant ID from context for authorization
	tenantID, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Failed to get tenant ID")
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	// Parse request body to session object
	var session types.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		logger.Error(ctx, "Failed to parse session data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Set session ID and tenant ID
	logger.Infof(ctx, "Updating session, ID: %s, tenant ID: %d", id, tenantID.(uint))
	session.ID = id
	session.TenantID = tenantID.(uint)

	// Call service to update session
	if err := h.sessionService.UpdateSession(ctx, &session); err != nil {
		if err == errors.ErrSessionNotFound {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return updated session
	logger.Infof(ctx, "Session updated successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    session,
	})
}

// DeleteSession deletes a session by its ID
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting session")

	// Get session ID from URL parameter
	id := c.Param("id")
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Call service to delete session
	logger.Infof(ctx, "Deleting session, ID: %s", id)
	if err := h.sessionService.DeleteSession(ctx, id); err != nil {
		if err == errors.ErrSessionNotFound {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return success message
	logger.Infof(ctx, "Session deleted successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session deleted successfully",
	})
}

// GenerateTitleRequest defines the request structure for generating a session title
type GenerateTitleRequest struct {
	Messages []types.Message `json:"messages" binding:"required"` // Messages to use as context for title generation
}

// GenerateTitle generates a title for a session based on message content
func (h *SessionHandler) GenerateTitle(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start generating session title")

	// Get session ID from URL parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Parse request body
	var request GenerateTitleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to parse request data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Get session from database
	session, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Call service to generate title
	logger.Infof(ctx, "Generating session title, session ID: %s, message count: %d", sessionID, len(request.Messages))
	title, err := h.sessionService.GenerateTitle(ctx, session, request.Messages)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return generated title
	logger.Infof(ctx, "Session title generated successfully, session ID: %s, title: %s", sessionID, title)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    title,
	})
}

// CreateKnowledgeQARequest defines the request structure for knowledge QA
type CreateKnowledgeQARequest struct {
	Query            string   `json:"query" binding:"required"` // Query text for knowledge base search
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`       // Selected knowledge base ID for this request
	AgentEnabled     bool     `json:"agent_enabled"`            // Whether agent mode is enabled for this request
}

// SearchKnowledgeRequest defines the request structure for searching knowledge without LLM summarization
type SearchKnowledgeRequest struct {
	Query           string `json:"query" binding:"required"`             // Query text to search for
	KnowledgeBaseID string `json:"knowledge_base_id" binding:"required"` // ID of the knowledge base to search
}

// SearchKnowledge performs knowledge base search without LLM summarization
func (h *SessionHandler) SearchKnowledge(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())

	logger.Info(ctx, "Start processing knowledge search request")

	// Parse request body
	var request SearchKnowledgeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to parse request data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Validate request parameters
	if request.Query == "" {
		logger.Error(ctx, "Query content is empty")
		c.Error(errors.NewBadRequestError("Query content cannot be empty"))
		return
	}

	if request.KnowledgeBaseID == "" {
		logger.Error(ctx, "Knowledge base ID is empty")
		c.Error(errors.NewBadRequestError("Knowledge base ID cannot be empty"))
		return
	}

	logger.Infof(
		ctx,
		"Knowledge search request, knowledge base ID: %s, query: %s",
		request.KnowledgeBaseID,
		request.Query,
	)

	// Directly call knowledge retrieval service without LLM summarization
	searchResults, err := h.sessionService.SearchKnowledge(ctx, request.KnowledgeBaseID, request.Query)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Knowledge search completed, found %d results", len(searchResults))

	// Return search results
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    searchResults,
	})
}

// ContinueStream handles continued streaming of an active response stream
func (h *SessionHandler) ContinueStream(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start continuing stream response processing")

	// Get session ID from URL parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Get message ID from query parameter
	messageID := c.Query("message_id")
	if messageID == "" {
		logger.Error(ctx, "Message ID is empty")
		c.Error(errors.NewBadRequestError("Missing message ID"))
		return
	}

	logger.Infof(ctx, "Continuing stream, session ID: %s, message ID: %s", sessionID, messageID)

	// Verify that the session exists and belongs to this tenant
	_, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		if err == errors.ErrSessionNotFound {
			logger.Warnf(ctx, "Session not found, ID: %s", sessionID)
			c.Error(errors.NewNotFoundError(err.Error()))
		} else {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	// Get the incomplete message
	message, err := h.messageService.GetMessage(ctx, sessionID, messageID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	if message == nil {
		logger.Warnf(ctx, "Incomplete message not found, session ID: %s, message ID: %s", sessionID, messageID)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Incomplete message not found",
		})
		return
	}

	// Get initial events from stream (offset 0)
	events, currentOffset, err := h.streamManager.GetEvents(ctx, sessionID, messageID, 0)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(fmt.Sprintf("Failed to get stream data: %s", err.Error())))
		return
	}

	if len(events) == 0 {
		logger.Warnf(ctx, "No events found in stream, session ID: %s, message ID: %s", sessionID, messageID)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No stream events found",
		})
		return
	}

	logger.Infof(
		ctx, "Preparing to replay %d events and continue streaming, session ID: %s, message ID: %s",
		len(events), sessionID, messageID,
	)

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Check if stream is already completed
	streamCompleted := false
	for _, evt := range events {
		if evt.Type == "complete" {
			streamCompleted = true
			break
		}
	}

	// Replay existing events
	logger.Debugf(ctx, "Replaying %d existing events", len(events))
	for _, evt := range events {
		response := &types.StreamResponse{
			ID:           message.RequestID,
			ResponseType: evt.Type,
			Content:      evt.Content,
			Done:         evt.Done,
			Data:         evt.Data,
		}

		// Special handling for references event
		if evt.Type == types.ResponseTypeReferences {
			if refs, ok := evt.Data["references"].(types.References); ok {
				response.KnowledgeReferences = refs
			}
		}

		c.SSEvent("message", response)
		c.Writer.Flush()
	}

	// If stream is already completed, send final event and return
	if streamCompleted {
		logger.Infof(ctx, "Stream already completed, session ID: %s, message ID: %s", sessionID, messageID)
		c.SSEvent("message", &types.StreamResponse{
			ID:           message.RequestID,
			ResponseType: types.ResponseTypeAnswer,
			Content:      "",
			Done:         true,
		})
		c.Writer.Flush()
		return
	}

	// Continue polling for new events
	logger.Debug(ctx, "Starting event update monitoring")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			logger.Debug(ctx, "Client connection closed")
			return

		case <-ticker.C:
			// Get new events from current offset
			newEvents, newOffset, err := h.streamManager.GetEvents(ctx, sessionID, messageID, currentOffset)
			if err != nil {
				logger.Errorf(ctx, "Failed to get new events: %v", err)
				return
			}

			// Send new events
			streamCompletedNow := false
			for _, evt := range newEvents {
				response := &types.StreamResponse{
					ID:           message.RequestID,
					ResponseType: evt.Type,
					Content:      evt.Content,
					Done:         evt.Done,
					Data:         evt.Data,
				}

				// Special handling for references event
				if evt.Type == types.ResponseTypeReferences {
					if refs, ok := evt.Data["references"].(types.References); ok {
						response.KnowledgeReferences = refs
					}
				}

				// Check for completion event
				if evt.Type == "complete" {
					streamCompletedNow = true
				}

				c.SSEvent("message", response)
				c.Writer.Flush()
			}

			// Update offset
			currentOffset = newOffset

			// If stream completed, send final event and exit
			if streamCompletedNow {
				logger.Infof(ctx, "Stream completed, session ID: %s, message ID: %s", sessionID, messageID)
				c.SSEvent("message", &types.StreamResponse{
					ID:           message.RequestID,
					ResponseType: types.ResponseTypeAnswer,
					Content:      "",
					Done:         true,
				})
				c.Writer.Flush()
				return
			}
		}
	}
}

// KnowledgeQA handles knowledge base question answering requests with LLM summarization
func (h *SessionHandler) KnowledgeQA(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())

	logger.Info(ctx, "Start processing knowledge QA request")

	// Get session ID from URL parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Parse request body
	var request CreateKnowledgeQARequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to parse request data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Create assistant message
	assistantMessage := &types.Message{
		SessionID:   sessionID,
		Role:        "assistant",
		RequestID:   c.GetString(types.RequestIDContextKey.String()),
		IsCompleted: false,
	}

	// Validate query content
	if request.Query == "" {
		logger.Error(ctx, "Query content is empty")
		c.Error(errors.NewBadRequestError("Query content cannot be empty"))
		return
	}

	logger.Infof(ctx, "Knowledge QA request, session ID: %s, query: %s", sessionID, request.Query)

	// Get session to prepare knowledge base IDs
	session, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get session, session ID: %s, error: %v", sessionID, err)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Prepare knowledge base IDs
	knowledgeBaseIDs := request.KnowledgeBaseIDs
	if len(knowledgeBaseIDs) == 0 && session.KnowledgeBaseID != "" {
		knowledgeBaseIDs = []string{session.KnowledgeBaseID}
		logger.Infof(ctx, "No knowledge base IDs in request, using session default: %s", session.KnowledgeBaseID)
	}

	// Use shared function to handle KnowledgeQA request
	h.handleKnowledgeQARequest(ctx, c, session, request.Query, knowledgeBaseIDs, assistantMessage, true)
}

// completeAssistantMessage marks an assistant message as complete and updates it
func (h *SessionHandler) completeAssistantMessage(ctx context.Context, assistantMessage *types.Message) {
	assistantMessage.UpdatedAt = time.Now()
	assistantMessage.IsCompleted = true
	_ = h.messageService.UpdateMessage(ctx, assistantMessage)
}

// handleKnowledgeQARequest handles a KnowledgeQA request with the given parameters
// This is a shared function used by both KnowledgeQA endpoint and AgentQA fallback
func (h *SessionHandler) handleKnowledgeQARequest(
	ctx context.Context,
	c *gin.Context,
	session *types.Session,
	query string,
	knowledgeBaseIDs []string,
	assistantMessage *types.Message,
	generateTitle bool, // Whether to generate title if session has no title
) {
	sessionID := session.ID
	requestID := c.GetString(types.RequestIDContextKey.String())

	// Create user message
	if _, err := h.messageService.CreateMessage(ctx, &types.Message{
		SessionID:   sessionID,
		Role:        "user",
		Content:     query,
		RequestID:   requestID,
		CreatedAt:   time.Now(),
		IsCompleted: true,
	}); err != nil {
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Create assistant message (response)
	assistantMessage.CreatedAt = time.Now()
	if _, err := h.messageService.CreateMessage(ctx, assistantMessage); err != nil {
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Validate knowledge bases
	if len(knowledgeBaseIDs) == 0 {
		logger.Error(ctx, "No knowledge base ID available")
		c.Error(errors.NewBadRequestError("At least one knowledge base ID is required"))
		return
	}

	logger.Infof(ctx, "Using knowledge bases: %v", knowledgeBaseIDs)

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Write initial agent_query event to StreamManager
	agentQueryEvent := interfaces.StreamEvent{
		ID:        fmt.Sprintf("query-%d", time.Now().UnixNano()),
		Type:      types.ResponseTypeAgentQuery,
		Content:   "",
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session_id":           sessionID,
			"assistant_message_id": assistantMessage.ID,
		},
	}
	if err := h.streamManager.AppendEvent(ctx, sessionID, assistantMessage.ID, agentQueryEvent); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"message_id": assistantMessage.ID,
		})
		// Non-fatal error, continue
	}

	// Create dedicated EventBus for this request
	eventBus := event.NewEventBus()
	// Create cancellable context for async operations
	asyncCtx, cancel := context.WithCancel(logger.CloneContext(ctx))

	// Register stop event handler to cancel the context
	eventBus.On(event.EventStop, func(ctx context.Context, evt event.Event) error {
		logger.Infof(ctx, "Received stop event, cancelling async operations for session: %s", sessionID)
		cancel()
		assistantMessage.Content = "用户停止了本次对话"
		h.completeAssistantMessage(ctx, assistantMessage)
		return nil
	})

	// Create stream handler with dedicated EventBus
	streamHandler := NewAgentStreamHandler(
		asyncCtx, sessionID, assistantMessage.ID, requestID,
		assistantMessage, h.streamManager, eventBus,
	)

	// Subscribe to events on the dedicated EventBus
	streamHandler.Subscribe()

	// Generate title if needed
	if generateTitle && session.Title == "" {
		logger.Infof(ctx, "Session has no title, starting async title generation, session ID: %s", sessionID)
		h.sessionService.GenerateTitleAsync(asyncCtx, session, query, eventBus)
	}

	eventBus.On(event.EventAgentFinalAnswer, func(ctx context.Context, evt event.Event) error {
		data, ok := evt.Data.(event.AgentFinalAnswerData)
		if !ok {
			return nil
		}
		assistantMessage.Content += data.Content
		if data.Done {
			logger.Infof(asyncCtx, "Knowledge QA service completed for session: %s", sessionID)
			h.completeAssistantMessage(asyncCtx, assistantMessage)
			cancel() // Clean up context
			return nil
		}
		return nil
	})

	// Call service to perform knowledge QA (async, emits events)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorWithFields(asyncCtx, errors.NewInternalServerError("Knowledge QA service panicked"), nil)
			}
		}()
		err := h.sessionService.KnowledgeQA(asyncCtx, session, query, knowledgeBaseIDs, assistantMessage.ID, eventBus)
		if err != nil {
			logger.ErrorWithFields(asyncCtx, err, nil)
			// Emit error event to dedicated EventBus
			eventBus.Emit(asyncCtx, event.Event{
				Type:      event.EventError,
				SessionID: sessionID,
				Data: event.ErrorData{
					Error:     err.Error(),
					Stage:     "knowledge_qa_execution",
					SessionID: sessionID,
				},
			})
			return
		}
	}()

	// Handle events for SSE (blocking until connection is done)
	h.handleAgentEventsForSSEWithHandler(ctx, c, sessionID, assistantMessage.ID, requestID, eventBus)
}

// AgentQA handles agent-based question answering with conversation history and streaming
func (h *SessionHandler) AgentQA(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	logger.Info(ctx, "Start processing agent QA request")

	// Get session ID from URL parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Parse request body
	var request CreateKnowledgeQARequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to parse request data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Validate query content
	if request.Query == "" {
		logger.Error(ctx, "Query content is empty")
		c.Error(errors.NewBadRequestError("Query content cannot be empty"))
		return
	}

	logger.Infof(ctx, "Agent QA request, session ID: %s, query: %s", sessionID, request.Query)

	// Get session information first
	session, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get session, session ID: %s, error: %v", sessionID, err)
		c.Error(errors.NewNotFoundError("Session not found"))
		return
	}

	// Create assistant message
	assistantMessage := &types.Message{
		SessionID:   sessionID,
		Role:        "assistant",
		RequestID:   c.GetString(types.RequestIDContextKey.String()),
		IsCompleted: false,
	}

	// Initialize AgentConfig if it doesn't exist
	if session.AgentConfig == nil {
		session.AgentConfig = &types.SessionAgentConfig{}
	}

	// Detect if knowledge bases or agent mode has changed
	knowledgeBasesChanged := false
	configChanged := false

	// Check if knowledge bases array has changed
	if len(request.KnowledgeBaseIDs) > 0 {
		// Compare arrays to detect changes
		currentKBs := session.AgentConfig.KnowledgeBases
		if len(currentKBs) != len(request.KnowledgeBaseIDs) {
			knowledgeBasesChanged = true
			configChanged = true
		} else {
			// Check if contents are different
			kbMap := make(map[string]bool)
			for _, kb := range currentKBs {
				kbMap[kb] = true
			}
			for _, kb := range request.KnowledgeBaseIDs {
				if !kbMap[kb] {
					knowledgeBasesChanged = true
					configChanged = true
					break
				}
			}
		}
		if knowledgeBasesChanged {
			logger.Infof(ctx, "Knowledge bases changed from %v to %v", session.AgentConfig.KnowledgeBases, request.KnowledgeBaseIDs)
		}
	}

	// Check if agent mode has changed
	currentAgentEnabled := session.AgentConfig.Enabled
	if request.AgentEnabled != currentAgentEnabled {
		logger.Infof(ctx, "Agent mode changed from %v to %v", currentAgentEnabled, request.AgentEnabled)
		configChanged = true
	}

	// If configuration changed, clear context and update session
	if configChanged {
		logger.Infof(ctx, "Configuration changed, clearing context for session: %s", sessionID)
		if knowledgeBasesChanged {
			// Clear the LLM context to prevent contamination
			if err := h.sessionService.ClearContext(ctx, sessionID); err != nil {
				logger.Errorf(ctx, "Failed to clear context for session %s: %v", sessionID, err)
				// Continue anyway - this is not a fatal error
			}
		}
		session.AgentConfig.KnowledgeBases = request.KnowledgeBaseIDs
		session.AgentConfig.Enabled = request.AgentEnabled
		// Persist the session changes
		if err := h.sessionService.UpdateSession(ctx, session); err != nil {
			logger.Errorf(ctx, "Failed to update session %s: %v", sessionID, err)
			c.Error(errors.NewInternalServerError("Failed to update session configuration"))
			return
		}
		logger.Infof(ctx, "Session configuration updated successfully for session: %s", sessionID)
	}

	// If Agent mode is disabled, delegate to KnowledgeQA
	if !request.AgentEnabled {
		logger.Infof(ctx, "Agent mode disabled, delegating to KnowledgeQA for session: %s", sessionID)

		// Use knowledge bases from request or session config
		knowledgeBaseIDs := request.KnowledgeBaseIDs
		if len(knowledgeBaseIDs) == 0 {
			knowledgeBaseIDs = session.AgentConfig.KnowledgeBases
		}

		// If still empty, use session default knowledge base
		if len(knowledgeBaseIDs) == 0 && session.KnowledgeBaseID != "" {
			knowledgeBaseIDs = []string{session.KnowledgeBaseID}
			logger.Infof(ctx, "Using session default knowledge base: %s", session.KnowledgeBaseID)
		}

		// Validate at least one knowledge base is available
		if len(knowledgeBaseIDs) == 0 {
			logger.Error(ctx, "No knowledge base available for delegation")
			c.Error(errors.NewBadRequestError("No knowledge base available. Please configure at least one knowledge base."))
			return
		}

		logger.Infof(ctx, "Delegating to KnowledgeQA with knowledge bases: %v", knowledgeBaseIDs)

		// Use shared function to handle KnowledgeQA request (no title generation for AgentQA fallback)
		h.handleKnowledgeQARequest(ctx, c, session, request.Query, knowledgeBaseIDs, assistantMessage, false)
		return
	}

	// Emit agent query event to create user message
	requestID := c.GetString(types.RequestIDContextKey.String())
	if err := event.Emit(ctx, event.Event{
		Type:      event.EventAgentQuery,
		SessionID: sessionID,
		RequestID: requestID,
		Data: event.AgentQueryData{
			SessionID: sessionID,
			Query:     request.Query,
			RequestID: requestID,
		},
	}); err != nil {
		logger.Errorf(ctx, "Failed to emit agent query event: %v", err)
		return
	}

	// Set headers for SSE immediately
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create user message
	if _, err := h.messageService.CreateMessage(ctx, &types.Message{
		SessionID:   sessionID,
		Role:        "user",
		Content:     request.Query,
		RequestID:   requestID,
		CreatedAt:   time.Now(),
		IsCompleted: true,
	}); err != nil {
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Create assistant message (response)
	assistantMessage.CreatedAt = time.Now()
	assistantMessagePtr, err := h.messageService.CreateMessage(ctx, assistantMessage)
	if err != nil {
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}
	assistantMessage = assistantMessagePtr

	logger.Infof(ctx, "Calling agent QA service, session ID: %s", sessionID)

	// SSE headers already set earlier when sending agent_query event
	// Write initial agent_query event to StreamManager
	agentQueryEvent := interfaces.StreamEvent{
		ID:        fmt.Sprintf("query-%d", time.Now().UnixNano()),
		Type:      types.ResponseTypeAgentQuery,
		Content:   "",
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session_id":           sessionID,
			"assistant_message_id": assistantMessage.ID,
		},
	}
	if err := h.streamManager.AppendEvent(ctx, sessionID, assistantMessage.ID, agentQueryEvent); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"message_id": assistantMessage.ID,
		})
		// Non-fatal error, continue
	}

	eventBus := event.NewEventBus()
	// Create cancellable context for async operations
	asyncCtx, cancel := context.WithCancel(logger.CloneContext(ctx))
	// Create stream handler with dedicated EventBus BEFORE calling AgentQA
	streamHandler := NewAgentStreamHandler(
		asyncCtx, sessionID, assistantMessage.ID, requestID,
		assistantMessage, h.streamManager, eventBus,
	)

	// Subscribe to events on the dedicated EventBus
	streamHandler.Subscribe()

	// Start async title generation if session has no title
	if session.Title == "" {
		logger.Infof(ctx, "Session has no title, starting async title generation, session ID: %s", sessionID)
		h.sessionService.GenerateTitleAsync(asyncCtx, session, request.Query, eventBus)
	}

	// Register stop event handler to cancel the context
	eventBus.On(event.EventStop, func(ctx context.Context, evt event.Event) error {
		logger.Warnf(asyncCtx, "Received stop event, cancelling async operations for session: %s", sessionID)
		cancel()
		assistantMessage.Content = "用户停止了本次对话"
		h.completeAssistantMessage(ctx, assistantMessage)
		return nil
	})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorWithFields(asyncCtx, errors.NewInternalServerError("Agent QA service panicked"), nil)
			}
			h.completeAssistantMessage(asyncCtx, assistantMessage)
			logger.Infof(asyncCtx, "Agent QA service completed for session: %s", sessionID)
		}()
		err := h.sessionService.AgentQA(asyncCtx, session, request.Query, assistantMessage.ID, eventBus)
		if err != nil {
			logger.ErrorWithFields(asyncCtx, err, nil)
			// Emit error event to dedicated EventBus
			eventBus.Emit(asyncCtx, event.Event{
				Type:      event.EventError,
				SessionID: sessionID,
				Data: event.ErrorData{
					Error:     err.Error(),
					Stage:     "agent_execution",
					SessionID: sessionID,
				},
			})
			return
		}
	}()

	// Handle events for SSE (blocking until connection is done)
	h.handleAgentEventsForSSEWithHandler(ctx, c, sessionID, assistantMessage.ID, requestID, eventBus)
}

// handleAgentEventsForSSEWithHandler handles agent events for SSE streaming using an existing handler
// The handler is already subscribed to events and AgentQA is already running
// This function polls StreamManager and pushes events to SSE, allowing graceful handling of disconnections
func (h *SessionHandler) handleAgentEventsForSSEWithHandler(
	ctx context.Context,
	c *gin.Context,
	sessionID, assistantMessageID, requestID string,
	eventBus *event.EventBus,
) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastOffset := 0
	log := logger.GetLogger(ctx)

	log.Infof("Starting pull-based SSE streaming for session=%s, message=%s", sessionID, assistantMessageID)

	for {
		select {
		case <-c.Request.Context().Done():
			// Connection closed, exit gracefully without panic
			log.Infof("Client disconnected, stopping SSE streaming for session=%s, message=%s", sessionID, assistantMessageID)
			return

		case <-ticker.C:
			// Get new events from StreamManager using offset
			events, newOffset, err := h.streamManager.GetEvents(ctx, sessionID, assistantMessageID, lastOffset)
			if err != nil {
				log.Warnf("Failed to get events from stream: %v", err)
				continue
			}

			// Send any new events
			streamCompleted := false
			for _, evt := range events {
				// Check for stop event
				if evt.Type == types.ResponseType(event.EventStop) {
					log.Infof("Detected stop event, triggering stop via EventBus for session=%s", sessionID)

					// Emit stop event to the EventBus to trigger context cancellation
					if eventBus != nil {
						eventBus.Emit(ctx, event.Event{
							Type:      event.EventStop,
							SessionID: sessionID,
							Data: event.StopData{
								SessionID: sessionID,
								MessageID: assistantMessageID,
								Reason:    "user_requested",
							},
						})
					}

					// Send stop notification to frontend
					c.SSEvent("message", &types.StreamResponse{
						ID:           requestID,
						ResponseType: "stop",
						Content:      "Generation stopped by user",
						Done:         true,
					})
					c.Writer.Flush()
					return
				}

				// Build StreamResponse from StreamEvent
				// For agent_query event, extract session_id and assistant_message_id from Data
				var sessionIDFromEvent, assistantMessageIDFromEvent string
				if evt.Type == types.ResponseTypeAgentQuery {
					if sid, ok := evt.Data["session_id"].(string); ok {
						sessionIDFromEvent = sid
					}
					if amid, ok := evt.Data["assistant_message_id"].(string); ok {
						assistantMessageIDFromEvent = amid
					}
				}
				response := &types.StreamResponse{
					ID:                 requestID,
					ResponseType:       evt.Type,
					Content:            evt.Content,
					Done:               evt.Done,
					Data:               evt.Data,
					SessionID:          sessionIDFromEvent,
					AssistantMessageID: assistantMessageIDFromEvent,
				}

				// Special handling for references event
				if evt.Type == types.ResponseTypeReferences {
					if refs, ok := evt.Data["references"].(types.References); ok {
						response.KnowledgeReferences = refs
					}
				}

				// Check for completion event
				if evt.Type == "complete" {
					streamCompleted = true
				}

				// Check if connection is still alive before writing
				if c.Request.Context().Err() != nil {
					log.Info("Connection closed during event sending, stopping")
					return
				}

				c.SSEvent("message", response)
				c.Writer.Flush()
			}

			// Update offset
			lastOffset = newOffset

			// Check if stream is completed
			if streamCompleted {
				log.Infof("Stream completed for session=%s, message=%s", sessionID, assistantMessageID)

				// Send final completion signal
				c.SSEvent("message", &types.StreamResponse{
					ID:           requestID,
					ResponseType: types.ResponseTypeAnswer,
					Content:      "",
					Done:         true,
				})
				c.Writer.Flush()
				return
			}
		}
	}
}

// StopSessionRequest represents the stop session request
type StopSessionRequest struct {
	MessageID string `json:"message_id" binding:"required"`
}

// StopSession handles the stop generation request
func (h *SessionHandler) StopSession(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	sessionID := c.Param("session_id")

	if sessionID == "" {
		c.JSON(400, gin.H{"error": "Session ID is required"})
		return
	}

	// Parse request body to get message_id
	var req StopSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
		})
		c.JSON(400, gin.H{"error": "message_id is required"})
		return
	}

	assistantMessageID := req.MessageID
	logger.Infof(ctx, "Stop generation request for session: %s, message: %s", sessionID, assistantMessageID)

	// Get tenant ID from context
	tenantID, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Failed to get tenant ID")
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	tenantIDUint := tenantID.(uint)

	// Verify message ownership and status
	message, err := h.messageService.GetMessage(ctx, sessionID, assistantMessageID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"message_id": assistantMessageID,
		})
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	// Verify message belongs to this session (double check)
	if message.SessionID != sessionID {
		logger.Warnf(ctx, "Message %s does not belong to session %s", assistantMessageID, sessionID)
		c.JSON(403, gin.H{"error": "Message does not belong to this session"})
		return
	}

	// Verify message belongs to the current tenant
	session, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
		})
		c.JSON(404, gin.H{"error": "Session not found"})
		return
	}

	if session.TenantID != tenantIDUint {
		logger.Warnf(ctx, "Session %s does not belong to tenant %d", sessionID, tenantIDUint)
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	// Check if message is already completed (stopped)
	if message.IsCompleted {
		logger.Infof(ctx, "Message %s is already completed, no need to stop", assistantMessageID)
		c.JSON(200, gin.H{
			"success": true,
			"message": "Message already completed",
		})
		return
	}

	// Write stop event to StreamManager for distributed support
	stopEvent := interfaces.StreamEvent{
		ID:        fmt.Sprintf("stop-%d", time.Now().UnixNano()),
		Type:      types.ResponseType(event.EventStop),
		Content:   "",
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session_id": sessionID,
			"message_id": assistantMessageID,
			"reason":     "user_requested",
		},
	}

	if err := h.streamManager.AppendEvent(ctx, sessionID, assistantMessageID, stopEvent); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"message_id": assistantMessageID,
		})
		c.JSON(500, gin.H{"error": "Failed to write stop event"})
		return
	}

	logger.Infof(ctx, "Stop event written successfully for session: %s, message: %s", sessionID, assistantMessageID)
	c.JSON(200, gin.H{
		"success": true,
		"message": "Generation stopped",
	})
}
