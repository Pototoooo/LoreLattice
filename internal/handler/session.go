package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// Get stream information
	streamInfo, err := h.streamManager.GetStream(ctx, sessionID, messageID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(fmt.Sprintf("Failed to get stream data: %s", err.Error())))
		return
	}

	if streamInfo == nil {
		logger.Warnf(ctx, "Active stream not found, session ID: %s, message ID: %s", sessionID, messageID)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Active stream not found",
		})
		return
	}

	// If stream is already completed, return the full message
	if streamInfo.IsCompleted {
		logger.Infof(
			ctx, "Stream already completed, returning directly, session ID: %s, message ID: %s", sessionID, messageID,
		)
		c.JSON(http.StatusOK, gin.H{
			"id":         message.ID,
			"role":       message.Role,
			"content":    message.Content,
			"created_at": message.CreatedAt,
			"done":       true,
		})
		return
	}

	logger.Infof(
		ctx, "Preparing to set SSE headers and send stream data, session ID: %s, message ID: %s", sessionID, messageID,
	)

	// Send knowledge references first if available
	if len(streamInfo.KnowledgeReferences) > 0 {
		logger.Debug(ctx, "Sending knowledge references")
		c.SSEvent("message", &types.StreamResponse{
			ID:                  message.RequestID,
			ResponseType:        types.ResponseTypeReferences,
			Done:                false,
			KnowledgeReferences: streamInfo.KnowledgeReferences,
		})
	}

	// Replay existing events
	if len(streamInfo.Events) > 0 {
		logger.Debugf(ctx, "Replaying %d existing events", len(streamInfo.Events))
		for _, evt := range streamInfo.Events {
			c.SSEvent("message", &types.StreamResponse{
				ID:           message.RequestID,
				ResponseType: evt.Type,
				Content:      evt.Content,
				Done:         evt.Done,
				Data:         evt.Data,
			})
		}
	}

	// Create channels to monitor event updates
	eventIndexCh := make(chan int, 10)
	doneCh := make(chan bool, 1)

	logger.Debug(ctx, "Starting event update monitoring")

	// Start a goroutine to monitor for new events
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		currentEventCount := len(streamInfo.Events)

		for {
			select {
			case <-ticker.C:
				latestStreamInfo, err := h.streamManager.GetStream(ctx, sessionID, messageID)
				if err != nil {
					logger.Errorf(ctx, "Failed to get stream data: %v", err)
					doneCh <- true
					return
				}

				if latestStreamInfo == nil {
					logger.Debug(ctx, "Stream no longer exists")
					doneCh <- true
					return
				}

				if latestStreamInfo.IsCompleted {
					logger.Debug(ctx, "Stream completed")
					doneCh <- true
					return
				}

				// Check for new events
				if len(latestStreamInfo.Events) > currentEventCount {
					eventIndexCh <- currentEventCount
					currentEventCount = len(latestStreamInfo.Events)
					logger.Debugf(ctx, "Detected %d new events", len(latestStreamInfo.Events)-currentEventCount)
				}

			case <-c.Request.Context().Done():
				logger.Debug(ctx, "Client connection closed")
				return
			}
		}
	}()

	logger.Info(ctx, "Starting stream response")

	// Stream new events to client
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			logger.Debug(ctx, "Client connection closed")
			return false

		case <-doneCh:
			logger.Debug(ctx, "Stream completed, sending completion notification")
			c.SSEvent("message", &types.StreamResponse{
				ID:           message.RequestID,
				ResponseType: types.ResponseTypeAnswer,
				Content:      "",
				Done:         true,
			})
			return false

		case startIdx := <-eventIndexCh:
			// Send new events
			latestStreamInfo, err := h.streamManager.GetStream(ctx, sessionID, messageID)
			if err == nil && latestStreamInfo != nil {
				for i := startIdx; i < len(latestStreamInfo.Events); i++ {
					evt := latestStreamInfo.Events[i]
					logger.Debugf(ctx, "Sending new event: %s", evt.Type)
					c.SSEvent("message", &types.StreamResponse{
						ID:           message.RequestID,
						ResponseType: evt.Type,
						Content:      evt.Content,
						Done:         evt.Done,
						Data:         evt.Data,
					})
				}
			}
			return true
		}
	})
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
	defer h.completeAssistantMessage(ctx, assistantMessage)

	// Validate query content
	if request.Query == "" {
		logger.Error(ctx, "Query content is empty")
		c.Error(errors.NewBadRequestError("Query content cannot be empty"))
		return
	}

	logger.Infof(ctx, "Knowledge QA request, session ID: %s, query: %s", sessionID, request.Query)

	// Get request ID for title generation
	requestID := c.GetString(types.RequestIDContextKey.String())

	// Get session to check if title needs to be generated
	session, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get session, session ID: %s, error: %v", sessionID, err)
		// Continue anyway - this is not critical
	} else {
		// Start async title generation if session has no title
		if session.Title == "" {
			logger.Infof(ctx, "Session has no title, starting async title generation, session ID: %s", sessionID)
			// Use a simple event bus for title generation
			titleEventBus := event.NewEventBus()
			// Subscribe to title events and push to SSE
			titleEventBus.On(event.EventSessionTitle, func(ctx context.Context, evt event.Event) error {
				data, ok := evt.Data.(event.SessionTitleData)
				if !ok {
					return nil
				}
				// Send title update via SSE
				c.SSEvent("message", &types.StreamResponse{
					ID:           requestID,
					ResponseType: types.ResponseTypeSessionTitle,
					Content:      data.Title,
					Done:         true,
					Data: map[string]interface{}{
						"session_id": data.SessionID,
						"title":      data.Title,
					},
				})
				c.Writer.Flush()
				return nil
			})
			h.sessionService.GenerateTitleAsync(ctx, session, request.Query, titleEventBus)
		}
	}

	// Create user message
	if _, err := h.messageService.CreateMessage(ctx, &types.Message{
		SessionID:   sessionID,
		Role:        "user",
		Content:     request.Query,
		RequestID:   c.GetString(types.RequestIDContextKey.String()),
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
	logger.Infof(ctx, "Calling knowledge QA service, session ID: %s", sessionID)

	// Prepare knowledge base IDs
	knowledgeBaseIDs := request.KnowledgeBaseIDs
	if len(knowledgeBaseIDs) == 0 && session.KnowledgeBaseID != "" {
		knowledgeBaseIDs = []string{session.KnowledgeBaseID}
		logger.Infof(ctx, "No knowledge base IDs in request, using session default: %s", session.KnowledgeBaseID)
	}

	// Validate knowledge bases
	if len(knowledgeBaseIDs) == 0 {
		logger.Error(ctx, "No knowledge base ID available")
		c.Error(errors.NewBadRequestError("At least one knowledge base ID is required"))
		return
	}

	logger.Infof(ctx, "Using knowledge bases: %v", knowledgeBaseIDs)

	// Call service to perform knowledge QA
	searchResults, respCh, err := h.sessionService.KnowledgeQA(ctx, session, request.Query, knowledgeBaseIDs)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}
	assistantMessage.KnowledgeReferences = searchResults

	// Register new stream with stream manager
	if err := h.streamManager.RegisterStream(ctx, sessionID, assistantMessage.ID, request.Query); err != nil {
		logger.GetLogger(ctx).Error("Register stream failed", "error", err)
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send knowledge references if available
	if len(searchResults) > 0 {
		logger.Debugf(ctx, "Sending reference content, total %d", len(searchResults))
		c.SSEvent("message", &types.StreamResponse{
			ID:                  requestID,
			ResponseType:        types.ResponseTypeReferences,
			KnowledgeReferences: searchResults,
		})
		c.Writer.Flush()
	} else {
		logger.Debug(ctx, "No reference content to send")
	}

	// Process streamed response
	func() {
		defer func() {
			// Mark stream as completed when done
			if err := h.streamManager.CompleteStream(ctx, sessionID, assistantMessage.ID); err != nil {
				logger.GetLogger(ctx).Error("Complete stream failed", "error", err)
			}
		}()
		for response := range respCh {
			response.ID = requestID
			c.SSEvent("message", response)
			c.Writer.Flush()
			if response.ResponseType == types.ResponseTypeAnswer {
				assistantMessage.Content += response.Content
				// Push event to stream for replay on refresh
				if err := h.streamManager.PushEvent(
					ctx, sessionID, assistantMessage.ID, interfaces.StreamEvent{
						ID:        uuid.New().String(),
						Type:      response.ResponseType,
						Content:   response.Content,
						Done:      response.Done,
						Timestamp: time.Now(),
					},
				); err != nil {
					logger.GetLogger(ctx).Error("Push answer event to stream failed", "error", err)
				}
			}
			if response.ResponseType == types.ResponseTypeReferences {
				// Update references
				if err := h.streamManager.UpdateReferences(
					ctx, sessionID, assistantMessage.ID, searchResults,
				); err != nil {
					logger.GetLogger(ctx).Error("Update stream references failed", "error", err)
				}
			}
		}
	}()
}

// completeAssistantMessage marks an assistant message as complete and updates it
func (h *SessionHandler) completeAssistantMessage(ctx context.Context, assistantMessage *types.Message) {
	assistantMessage.UpdatedAt = time.Now()
	assistantMessage.IsCompleted = true
	_ = h.messageService.UpdateMessage(ctx, assistantMessage)
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

	// Create assistant message
	assistantMessage := &types.Message{
		SessionID:   sessionID,
		Role:        "assistant",
		RequestID:   c.GetString(types.RequestIDContextKey.String()),
		IsCompleted: false,
	}
	defer h.completeAssistantMessage(ctx, assistantMessage)

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

		// Create user message
		requestID := c.GetString(types.RequestIDContextKey.String())
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
		if _, err := h.messageService.CreateMessage(ctx, assistantMessage); err != nil {
			c.Error(errors.NewInternalServerError(err.Error()))
			return
		}

		// Call KnowledgeQA service
		searchResults, respCh, err := h.sessionService.KnowledgeQA(ctx, session, request.Query, knowledgeBaseIDs)
		if err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError(err.Error()))
			return
		}
		assistantMessage.KnowledgeReferences = searchResults

		// Register new stream with stream manager
		if err := h.streamManager.RegisterStream(ctx, sessionID, assistantMessage.ID, request.Query); err != nil {
			logger.GetLogger(ctx).Error("Register stream failed", "error", err)
		}

		// Set headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// Send knowledge references if available
		if len(searchResults) > 0 {
			logger.Debugf(ctx, "Sending reference content, total %d", len(searchResults))
			c.SSEvent("message", &types.StreamResponse{
				ID:                  requestID,
				ResponseType:        types.ResponseTypeReferences,
				KnowledgeReferences: searchResults,
			})
			c.Writer.Flush()
		} else {
			logger.Debug(ctx, "No reference content to send")
		}

		// Process streamed response
		func() {
			defer func() {
				// Mark stream as completed when done
				if err := h.streamManager.CompleteStream(ctx, sessionID, assistantMessage.ID); err != nil {
					logger.GetLogger(ctx).Error("Complete stream failed", "error", err)
				}
			}()
			for response := range respCh {
				response.ID = requestID
				c.SSEvent("message", response)
				c.Writer.Flush()
				if response.ResponseType == types.ResponseTypeAnswer {
					assistantMessage.Content += response.Content
				}
			}
			assistantMessage.IsCompleted = true
		}()

		// Update message with final content and references
		assistantMessage.UpdatedAt = time.Now()
		if err := h.messageService.UpdateMessage(ctx, assistantMessage); err != nil {
			logger.Errorf(ctx, "Failed to update assistant message: %v", err)
		}

		logger.Infof(ctx, "KnowledgeQA delegation completed for session: %s", sessionID)
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

	// Send agent query event to frontend via SSE to trigger loading state immediately
	c.SSEvent("message", &types.StreamResponse{
		ID:           requestID,
		ResponseType: types.ResponseTypeAgentQuery,
		Content:      "Agent query processing started",
		Done:         false,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"query":      request.Query,
			"request_id": requestID,
		},
	})
	c.Writer.Flush()

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

	// Register new stream with stream manager
	if err := h.streamManager.RegisterStream(ctx, sessionID, assistantMessage.ID, request.Query); err != nil {
		logger.GetLogger(ctx).Error("Register stream failed", "error", err)
	}

	// SSE headers already set earlier when sending agent_query event

	eventBus := event.NewEventBus()
	// Create stream handler with dedicated EventBus BEFORE calling AgentQA
	streamHandler := NewAgentStreamHandler(
		ctx, c, sessionID, assistantMessage.ID, requestID,
		assistantMessage, h.streamManager, eventBus,
	)

	// Subscribe to events on the dedicated EventBus
	streamHandler.Subscribe()

	// Start async title generation if session has no title
	if session.Title == "" {
		logger.Infof(ctx, "Session has no title, starting async title generation, session ID: %s", sessionID)
		// Subscribe to title events on the dedicated EventBus
		eventBus.On(event.EventSessionTitle, func(ctx context.Context, evt event.Event) error {
			data, ok := evt.Data.(event.SessionTitleData)
			if !ok {
				return nil
			}
			// Send title update via SSE
			c.SSEvent("message", &types.StreamResponse{
				ID:           requestID,
				ResponseType: types.ResponseTypeSessionTitle,
				Content:      data.Title,
				Done:         true,
				Data: map[string]interface{}{
					"session_id": data.SessionID,
					"title":      data.Title,
				},
			})
			c.Writer.Flush()
			return nil
		})
		h.sessionService.GenerateTitleAsync(ctx, session, request.Query, eventBus)
	}

	// Call service to perform agent QA
	go func() {
		searchResults, err := h.sessionService.AgentQA(ctx, session, request.Query, assistantMessage.ID, eventBus)
		if err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			// Emit error event to dedicated EventBus
			eventBus.Emit(ctx, event.Event{
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
		assistantMessage.KnowledgeReferences = searchResults
	}()

	// Handle events for SSE (blocking until connection is done)
	h.handleAgentEventsForSSEWithHandler(ctx, c, sessionID, assistantMessage.ID)
}

// handleAgentEventsForSSEWithHandler handles agent events for SSE streaming using an existing handler
// The handler is already subscribed to events and AgentQA is already running
func (h *SessionHandler) handleAgentEventsForSSEWithHandler(
	ctx context.Context,
	c *gin.Context,
	sessionID, assistantMessageID string,
) {
	// Wait for completion - events are already being handled by streamHandler
	// The connection will be closed when the gin context is done
	<-c.Request.Context().Done()

	// Complete stream when done
	if err := h.streamManager.CompleteStream(ctx, sessionID, assistantMessageID); err != nil {
		logger.GetLogger(ctx).Error("Complete stream failed", "error", err)
	}
}
