package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	chatpipline "github.com/Tencent/WeKnora/internal/application/service/chat_pipline"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/tracing"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// sessionService implements the SessionService interface for managing conversation sessions
type sessionService struct {
	cfg                  *config.Config                  // Application configuration
	sessionRepo          interfaces.SessionRepository    // Repository for session data
	messageRepo          interfaces.MessageRepository    // Repository for message data
	knowledgeBaseService interfaces.KnowledgeBaseService // Service for knowledge base operations
	modelService         interfaces.ModelService         // Service for model operations
	tenantService        interfaces.TenantService        // Service for tenant operations
	eventManager         *chatpipline.EventManager       // Event manager for chat pipeline
	agentService         interfaces.AgentService         // Service for agent operations
	contextManager       interfaces.ContextManager       // LLM context manager (separate from message storage)
}

// NewSessionService creates a new session service instance with all required dependencies
func NewSessionService(cfg *config.Config,
	sessionRepo interfaces.SessionRepository,
	messageRepo interfaces.MessageRepository,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	modelService interfaces.ModelService,
	tenantService interfaces.TenantService,
	eventManager *chatpipline.EventManager,
	agentService interfaces.AgentService,
) interfaces.SessionService {
	// Create default context manager with sliding window strategy
	// Default: 8192 tokens max, keep last 20 messages
	defaultStrategy := NewSlidingWindowStrategy(20)
	contextManager := NewContextManager(defaultStrategy, 8192)

	return &sessionService{
		cfg:                  cfg,
		sessionRepo:          sessionRepo,
		messageRepo:          messageRepo,
		knowledgeBaseService: knowledgeBaseService,
		modelService:         modelService,
		tenantService:        tenantService,
		eventManager:         eventManager,
		agentService:         agentService,
		contextManager:       contextManager,
	}
}

// CreateSession creates a new conversation session
func (s *sessionService) CreateSession(ctx context.Context, session *types.Session) (*types.Session, error) {
	logger.Info(ctx, "Start creating session")

	// Validate tenant ID
	if session.TenantID == 0 {
		logger.Error(ctx, "Failed to create session: tenant ID cannot be empty")
		return nil, errors.New("tenant ID is required")
	}

	logger.Infof(ctx, "Creating session, tenant ID: %d, model ID: %s, knowledge base ID: %s",
		session.TenantID, session.SummaryModelID, session.KnowledgeBaseID)

	// Create session in repository
	createdSession, err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	logger.Infof(ctx, "Session created successfully, ID: %s, tenant ID: %d", createdSession.ID, createdSession.TenantID)
	return createdSession, nil
}

// GetSession retrieves a session by its ID
func (s *sessionService) GetSession(ctx context.Context, id string) (*types.Session, error) {
	logger.Info(ctx, "Start retrieving session")

	// Validate session ID
	if id == "" {
		logger.Error(ctx, "Failed to get session: session ID cannot be empty")
		return nil, errors.New("session id is required")
	}

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Retrieving session, ID: %s, tenant ID: %d", id, tenantID)

	// Get session from repository
	session, err := s.sessionRepo.Get(ctx, tenantID, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": id,
			"tenant_id":  tenantID,
		})
		return nil, err
	}

	logger.Infof(ctx, "Session retrieved successfully, ID: %s, tenant ID: %d", session.ID, session.TenantID)
	return session, nil
}

// GetSessionsByTenant retrieves all sessions for the current tenant
func (s *sessionService) GetSessionsByTenant(ctx context.Context) ([]*types.Session, error) {
	logger.Info(ctx, "Start retrieving all sessions for tenant")

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Retrieving all sessions for tenant, tenant ID: %d", tenantID)

	// Get sessions from repository
	sessions, err := s.sessionRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": tenantID,
		})
		return nil, err
	}

	logger.Infof(
		ctx, "Tenant sessions retrieved successfully, tenant ID: %d, session count: %d", tenantID, len(sessions),
	)
	return sessions, nil
}

// GetPagedSessionsByTenant retrieves sessions for the current tenant with pagination
func (s *sessionService) GetPagedSessionsByTenant(ctx context.Context,
	pagination *types.Pagination,
) (*types.PageResult, error) {
	logger.Info(ctx, "Start retrieving paged sessions for tenant")

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Retrieving paged sessions for tenant, tenant ID: %d, page: %d, page size: %d",
		tenantID, pagination.Page, pagination.PageSize)

	// Get paged sessions from repository
	sessions, total, err := s.sessionRepo.GetPagedByTenantID(ctx, tenantID, pagination)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": tenantID,
			"page":      pagination.Page,
			"page_size": pagination.PageSize,
		})
		return nil, err
	}

	logger.Infof(ctx, "Tenant paged sessions retrieved successfully, tenant ID: %d, total: %d", tenantID, total)
	return types.NewPageResult(total, pagination, sessions), nil
}

// UpdateSession updates an existing session's properties
func (s *sessionService) UpdateSession(ctx context.Context, session *types.Session) error {
	logger.Info(ctx, "Start updating session")

	// Validate session ID
	if session.ID == "" {
		logger.Error(ctx, "Failed to update session: session ID cannot be empty")
		return errors.New("session id is required")
	}

	logger.Infof(ctx, "Updating session, ID: %s, tenant ID: %d", session.ID, session.TenantID)

	// Update session in repository
	err := s.sessionRepo.Update(ctx, session)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": session.ID,
			"tenant_id":  session.TenantID,
		})
		return err
	}

	logger.Infof(ctx, "Session updated successfully, ID: %s", session.ID)
	return nil
}

// DeleteSession removes a session by its ID
func (s *sessionService) DeleteSession(ctx context.Context, id string) error {
	logger.Info(ctx, "Start deleting session")

	// Validate session ID
	if id == "" {
		logger.Error(ctx, "Failed to delete session: session ID cannot be empty")
		return errors.New("session id is required")
	}

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Deleting session, ID: %s, tenant ID: %d", id, tenantID)

	// Delete session from repository
	err := s.sessionRepo.Delete(ctx, tenantID, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": id,
			"tenant_id":  tenantID,
		})
		return err
	}

	logger.Infof(ctx, "Session deleted successfully, ID: %s", id)
	return nil
}

// GenerateTitle generates a title for the current conversation content
func (s *sessionService) GenerateTitle(ctx context.Context,
	sessionID string, messages []types.Message,
) (string, error) {
	logger.Info(ctx, "Start generating session title")

	// Validate session ID
	if sessionID == "" {
		logger.Error(ctx, "Failed to generate title: session ID cannot be empty")
		return "", errors.New("session id is required")
	}

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Getting session info, session ID: %s, tenant ID: %d", sessionID, tenantID)

	// Get session from repository
	session, err := s.sessionRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"tenant_id":  tenantID,
		})
		return "", err
	}

	// Skip if title already exists
	if session.Title != "" {
		logger.Infof(ctx, "Session already has a title, session ID: %s, title: %s", sessionID, session.Title)
		return session.Title, nil
	}

	// Get the first user message, either from provided messages or repository
	var message *types.Message
	if len(messages) == 0 {
		logger.Info(ctx, "Message list is empty, getting the first user message")
		message, err = s.messageRepo.GetFirstMessageOfUser(ctx, sessionID)
		if err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{
				"session_id": sessionID,
			})
			return "", err
		}
	} else {
		logger.Info(ctx, "Searching for user message in message list")
		for _, m := range messages {
			if m.Role == "user" {
				message = &m
				break
			}
		}
	}

	// Ensure a user message was found
	if message == nil {
		logger.Error(ctx, "No user message found, cannot generate title")
		return "", errors.New("no user message found")
	}

	// Get chat model
	logger.Infof(ctx, "Getting chat model, model ID: %s", session.SummaryModelID)
	chatModel, err := s.modelService.GetChatModel(ctx, session.SummaryModelID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"model_id": session.SummaryModelID,
		})
		return "", err
	}

	// Prepare messages for title generation
	logger.Info(ctx, "Preparing to generate session title")
	var chatMessages []chat.Message
	chatMessages = append(chatMessages,
		chat.Message{Role: "system", Content: s.cfg.Conversation.GenerateSessionTitlePrompt},
	)
	chatMessages = append(chatMessages,
		chat.Message{Role: "user", Content: message.Content + " /no_think"},
	)

	// Call model to generate title
	thinking := false
	logger.Info(ctx, "Calling model to generate title")
	response, err := chatModel.Chat(ctx, chatMessages, &chat.ChatOptions{
		Temperature: 0.3,
		Thinking:    &thinking,
	})
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		return "", err
	}

	// Process and store the generated title
	session.Title = strings.TrimPrefix(response.Content, "<think>\n\n</think>")
	logger.Infof(ctx, "Title generated successfully: %s", session.Title)

	// Update session with new title
	logger.Info(ctx, "Updating session title")
	err = s.sessionRepo.Update(ctx, session)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		return "", err
	}

	logger.Infof(ctx, "Session title updated successfully, ID: %s, title: %s", sessionID, session.Title)
	return session.Title, nil
}

// KnowledgeQA performs knowledge base question answering with LLM summarization
func (s *sessionService) KnowledgeQA(ctx context.Context, sessionID, query string) (
	[]*types.SearchResult, <-chan types.StreamResponse, error,
) {
	logger.Info(ctx, "Start knowledge base question answering")
	logger.Infof(ctx, "Knowledge base question answering parameters, session ID: %s, query: %s", sessionID, query)

	// Get tenant ID from context
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Getting session info, session ID: %s, tenant ID: %d", sessionID, tenantID)

	// Get session information
	session, err := s.sessionRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get session, session ID: %s, error: %v", sessionID, err)
		return nil, nil, err
	}

	// Validate knowledge base association
	if session.KnowledgeBaseID == "" {
		logger.Warnf(ctx, "Session has no associated knowledge base, session ID: %s", sessionID)
		return nil, nil, errors.New("session has no knowledge base")
	}

	// Create chat management object with session settings
	logger.Infof(ctx, "Creating chat manage object, knowledge base ID: %s", session.KnowledgeBaseID)
	chatManage := &types.ChatManage{
		Query:            query,
		RewriteQuery:     query,
		SessionID:        sessionID,
		KnowledgeBaseID:  session.KnowledgeBaseID,
		VectorThreshold:  session.VectorThreshold,
		KeywordThreshold: session.KeywordThreshold,
		EmbeddingTopK:    session.EmbeddingTopK,
		RerankModelID:    session.RerankModelID,
		RerankTopK:       session.RerankTopK,
		RerankThreshold:  session.RerankThreshold,
		ChatModelID:      session.SummaryModelID,
		SummaryConfig: types.SummaryConfig{
			MaxTokens:           session.SummaryParameters.MaxTokens,
			RepeatPenalty:       session.SummaryParameters.RepeatPenalty,
			TopK:                session.SummaryParameters.TopK,
			TopP:                session.SummaryParameters.TopP,
			FrequencyPenalty:    session.SummaryParameters.FrequencyPenalty,
			PresencePenalty:     session.SummaryParameters.PresencePenalty,
			Prompt:              session.SummaryParameters.Prompt,
			ContextTemplate:     session.SummaryParameters.ContextTemplate,
			Temperature:         session.SummaryParameters.Temperature,
			Seed:                session.SummaryParameters.Seed,
			NoMatchPrefix:       session.SummaryParameters.NoMatchPrefix,
			MaxCompletionTokens: session.SummaryParameters.MaxCompletionTokens,
		},
		FallbackResponse: session.FallbackResponse,
	}

	// Start knowledge QA event processing
	logger.Info(ctx, "Triggering knowledge base question answering event")
	err = s.KnowledgeQAByEvent(ctx, chatManage, types.Pipline["rag_stream"])
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id":        sessionID,
			"knowledge_base_id": session.KnowledgeBaseID,
		})
		return nil, nil, err
	}

	logger.Info(ctx, "Knowledge base question answering completed")
	return chatManage.MergeResult, chatManage.ResponseChan, nil
}

// KnowledgeQAByEvent processes knowledge QA through a series of events in the pipeline
func (s *sessionService) KnowledgeQAByEvent(ctx context.Context,
	chatManage *types.ChatManage, eventList []types.EventType,
) error {
	ctx, span := tracing.ContextWithSpan(ctx, "SessionService.KnowledgeQAByEvent")
	defer span.End()

	logger.Info(ctx, "Start processing knowledge base question answering through events")
	logger.Infof(ctx, "Knowledge base question answering parameters, session ID: %s, knowledge base ID: %s, query: %s",
		chatManage.SessionID, chatManage.KnowledgeBaseID, chatManage.Query)

	// Prepare method list for logging and tracing
	methods := []string{}
	for _, event := range eventList {
		methods = append(methods, string(event))
	}

	// Set up tracing attributes
	logger.Infof(ctx, "Trigger event list: %v", methods)
	span.SetAttributes(
		attribute.String("request_id", ctx.Value(types.RequestIDContextKey).(string)),
		attribute.String("query", chatManage.Query),
		attribute.String("method", strings.Join(methods, ",")),
	)

	// Process each event in sequence
	for _, event := range eventList {
		logger.Infof(ctx, "Starting to trigger event: %v", event)
		err := s.eventManager.Trigger(ctx, event, chatManage)

		// Handle case where search returns no results
		if err == chatpipline.ErrSearchNothing {
			logger.Warnf(ctx, "Event %v triggered, search result is empty, using fallback response", event)
			chatManage.ResponseChan = chatpipline.NewFallbackChan(ctx, chatManage.FallbackResponse)
			chatManage.ChatResponse = &types.ChatResponse{Content: chatManage.FallbackResponse}
			return nil
		}

		// Handle other errors
		if err != nil {
			logger.Errorf(ctx, "Event triggering failed, event: %v, error type: %s, description: %s, error: %v",
				event, err.ErrorType, err.Description, err.Err)
			span.RecordError(err.Err)
			span.SetStatus(codes.Error, err.Description)
			span.SetAttributes(attribute.String("error_type", err.ErrorType))
			return err.Err
		}
		logger.Infof(ctx, "Event %v triggered successfully", event)
	}

	logger.Info(ctx, "All events triggered successfully")
	return nil
}

// SearchKnowledge performs knowledge base search without LLM summarization
func (s *sessionService) SearchKnowledge(ctx context.Context,
	knowledgeBaseID, query string,
) ([]*types.SearchResult, error) {
	logger.Info(ctx, "Start knowledge base search without LLM summary")
	logger.Infof(ctx, "Knowledge base search parameters, knowledge base ID: %s, query: %s", knowledgeBaseID, query)

	// Create default retrieval parameters
	chatManage := &types.ChatManage{
		Query:            query,
		RewriteQuery:     query,
		KnowledgeBaseID:  knowledgeBaseID,
		VectorThreshold:  s.cfg.Conversation.VectorThreshold,  // Use default configuration
		KeywordThreshold: s.cfg.Conversation.KeywordThreshold, // Use default configuration
		EmbeddingTopK:    s.cfg.Conversation.EmbeddingTopK,    // Use default configuration
		RerankTopK:       s.cfg.Conversation.RerankTopK,       // Use default configuration
		RerankThreshold:  s.cfg.Conversation.RerankThreshold,  // Use default configuration
	}

	// Get default models
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to get models: %v", err)
		return nil, err
	}

	// Find the first available rerank model
	for _, model := range models {
		if model.Type == types.ModelTypeRerank {
			chatManage.RerankModelID = model.ID
			break
		}
	}

	// Use specific event list, only including retrieval-related events, not LLM summarization
	searchEvents := []types.EventType{
		types.PREPROCESS_QUERY, // Preprocess query
		types.CHUNK_SEARCH,     // Vector search
		types.CHUNK_RERANK,     // Rerank search results
		types.CHUNK_MERGE,      // Merge search results
		types.FILTER_TOP_K,     // Filter top K results
	}

	ctx, span := tracing.ContextWithSpan(ctx, "SessionService.SearchKnowledge")
	defer span.End()

	// Prepare method list for logging and tracing
	methods := []string{}
	for _, event := range searchEvents {
		methods = append(methods, string(event))
	}

	// Set up tracing attributes
	logger.Infof(ctx, "Trigger search event list: %v", methods)
	span.SetAttributes(
		attribute.String("query", query),
		attribute.String("knowledge_base_id", knowledgeBaseID),
		attribute.String("method", strings.Join(methods, ",")),
	)

	// Process each search event in sequence
	for _, event := range searchEvents {
		logger.Infof(ctx, "Starting to trigger search event: %v", event)
		err := s.eventManager.Trigger(ctx, event, chatManage)

		// Handle case where search returns no results
		if err == chatpipline.ErrSearchNothing {
			logger.Warnf(ctx, "Event %v triggered, search result is empty", event)
			return []*types.SearchResult{}, nil
		}

		// Handle other errors
		if err != nil {
			logger.Errorf(ctx, "Event triggering failed, event: %v, error type: %s, description: %s, error: %v",
				event, err.ErrorType, err.Description, err.Err)
			span.RecordError(err.Err)
			span.SetStatus(codes.Error, err.Description)
			span.SetAttributes(attribute.String("error_type", err.ErrorType))
			return nil, err.Err
		}
		logger.Infof(ctx, "Event %v triggered successfully", event)
	}

	logger.Infof(ctx, "Knowledge base search completed, found %d results", len(chatManage.MergeResult))
	return chatManage.MergeResult, nil
}

// AgentQA performs agent-based question answering with conversation history and streaming support
func (s *sessionService) AgentQA(ctx context.Context, session *types.Session, query string, assistantMessageID string, eventBus *event.EventBus) (
	[]*types.SearchResult, error,
) {
	sessionID := session.ID
	tenantID := ctx.Value(types.TenantIDContextKey).(uint)
	logger.Infof(ctx, "Start agent-based question answering, session ID: %s, tenant ID: %d, query: %s", sessionID, tenantID, query)

	// Get effective agent configuration (session > tenant)
	var agentConfig *types.AgentConfig

	// Fall back to tenant-level agent config (global default)
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if tenantInfo.AgentConfig != nil && tenantInfo.AgentConfig.Enabled {
		logger.Infof(ctx, "Using tenant-level agent config for tenant: %d", tenantInfo.ID)
		agentConfig = tenantInfo.AgentConfig
	}

	// Check if agent is enabled (either from session or tenant)
	if agentConfig == nil || !agentConfig.Enabled {
		logger.Warnf(ctx, "Agent not enabled for session: %s (neither session nor tenant has agent config)", sessionID)
		return nil, errors.New("agent not enabled for this session")
	}

	// Set knowledge bases for agent if not already configured
	// Priority: AgentConfig.KnowledgeBases > Session.KnowledgeBaseID > All tenant knowledge bases
	if len(agentConfig.KnowledgeBases) == 0 {
		if session.KnowledgeBaseID != "" {
			// Use session's knowledge base as fallback
			agentConfig.KnowledgeBases = []string{session.KnowledgeBaseID}
			logger.Infof(ctx, "Using session's knowledge base for agent: %s", session.KnowledgeBaseID)
		} else {
			// Default to all knowledge bases under the tenant
			logger.Infof(ctx, "No knowledge bases specified, fetching all knowledge bases for tenant")
			allKBs, err := s.knowledgeBaseService.ListKnowledgeBases(ctx)
			if err != nil {
				logger.Errorf(ctx, "Failed to list knowledge bases for tenant: %v", err)
				return nil, fmt.Errorf("failed to list knowledge bases: %w", err)
			}

			if len(allKBs) == 0 {
				logger.Warnf(ctx, "No knowledge bases available for agent session: %s", sessionID)
				return nil, errors.New("no knowledge bases available for agent")
			}

			// Extract knowledge base IDs
			agentConfig.KnowledgeBases = make([]string, len(allKBs))
			for i, kb := range allKBs {
				agentConfig.KnowledgeBases[i] = kb.ID
			}
			logger.Infof(ctx, "Agent defaulting to all %d knowledge base(s) in tenant: %v",
				len(agentConfig.KnowledgeBases), agentConfig.KnowledgeBases)
		}
	} else {
		logger.Infof(ctx, "Agent configured with %d knowledge base(s): %v", len(agentConfig.KnowledgeBases), agentConfig.KnowledgeBases)
	}

	// Set ThinkingModelID from session's SummaryModelID if not already set
	if agentConfig.ThinkingModelID == "" && session.SummaryModelID != "" {
		agentConfig.ThinkingModelID = session.SummaryModelID
		logger.Infof(ctx, "Using session's SummaryModelID as ThinkingModelID: %s", session.SummaryModelID)
	}

	// Create agent engine with EventBus
	logger.Info(ctx, "Creating agent engine")
	engine, err := s.agentService.CreateAgentEngine(ctx, agentConfig, eventBus)
	if err != nil {
		logger.Errorf(ctx, "Failed to create agent engine: %v", err)
		return nil, err
	}

	// Get LLM context from context manager
	llmContext, err := s.getContextForSession(ctx, session, sessionID)
	if err != nil {
		logger.Warnf(ctx, "Failed to get LLM context: %v, continuing without history", err)
		llmContext = []chat.Message{}
	}
	logger.Infof(ctx, "Loaded %d messages from LLM context manager", len(llmContext))

	// Execute agent with streaming (asynchronously)
	// Events will be emitted to EventBus and handled by the Handler layer
	logger.Info(ctx, "Executing agent with streaming")
	go func() {
		if _, err := engine.Execute(ctx, sessionID, query, llmContext); err != nil {
			logger.Errorf(ctx, "Agent execution failed: %v", err)
			// Emit error event to the EventBus used by this agent
			eventBus.Emit(ctx, event.Event{
				Type:      event.EventError,
				SessionID: sessionID,
				Data: event.ErrorData{
					Error:     err.Error(),
					Stage:     "agent_execution",
					SessionID: sessionID,
				},
			})
		}
	}()

	// Return empty - events will be handled by Handler via EventBus subscription
	return nil, nil
}

// getContextForSession retrieves LLM context for a session
// Uses context manager which handles token limits and compression automatically
// This is separate from the message storage/conversation history
func (s *sessionService) getContextForSession(ctx context.Context, session *types.Session, sessionID string) ([]chat.Message, error) {
	// Check if session has custom context configuration
	var contextManager interfaces.ContextManager

	if session.ContextConfig != nil && session.ContextConfig.Enabled {
		// Create custom context manager based on session configuration
		logger.Infof(ctx, "Using custom context config for session %s: strategy=%s, max_tokens=%d, recent_count=%d",
			sessionID, session.ContextConfig.CompressionStrategy, session.ContextConfig.MaxTokens, session.ContextConfig.RecentMessageCount)

		var strategy interfaces.CompressionStrategy
		switch session.ContextConfig.CompressionStrategy {
		case types.ContextCompressionSlidingWindow:
			strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
		case types.ContextCompressionSmart:
			// For smart compression, we need the chat model
			chatModel, err := s.modelService.GetChatModel(ctx, session.SummaryModelID)
			if err != nil {
				logger.Warnf(ctx, "Failed to get chat model for smart compression, falling back to sliding window: %v", err)
				strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
			} else {
				strategy = NewSmartCompressionStrategy(session.ContextConfig.RecentMessageCount, chatModel, 5)
			}
		default:
			logger.Warnf(ctx, "Unknown compression strategy %s, using sliding window", session.ContextConfig.CompressionStrategy)
			strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
		}

		contextManager = NewContextManager(strategy, session.ContextConfig.MaxTokens)
	} else {
		// Use default context manager
		logger.Debugf(ctx, "Using default context manager for session %s", sessionID)
		contextManager = s.contextManager
	}

	// Get context from the context manager
	history, err := contextManager.GetContext(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	// Log context statistics
	stats, _ := contextManager.GetContextStats(ctx, sessionID)
	if stats != nil {
		logger.Infof(ctx, "LLM context stats for session %s: messages=%d, tokens=~%d, compressed=%v",
			sessionID, stats.MessageCount, stats.TokenCount, stats.IsCompressed)
	}

	return history, nil
}

// AddMessageToContext adds a message to the LLM context
// This should be called after saving a message to the database
// The context manager handles token limits and compression automatically
func (s *sessionService) AddMessageToContext(ctx context.Context, session *types.Session, sessionID string, message chat.Message) error {
	// Determine which context manager to use
	var contextManager interfaces.ContextManager

	if session.ContextConfig != nil && session.ContextConfig.Enabled {
		// Create custom context manager based on session configuration
		var strategy interfaces.CompressionStrategy
		switch session.ContextConfig.CompressionStrategy {
		case types.ContextCompressionSlidingWindow:
			strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
		case types.ContextCompressionSmart:
			chatModel, err := s.modelService.GetChatModel(ctx, session.SummaryModelID)
			if err != nil {
				strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
			} else {
				strategy = NewSmartCompressionStrategy(session.ContextConfig.RecentMessageCount, chatModel, 5)
			}
		default:
			strategy = NewSlidingWindowStrategy(session.ContextConfig.RecentMessageCount)
		}
		contextManager = NewContextManager(strategy, session.ContextConfig.MaxTokens)
	} else {
		contextManager = s.contextManager
	}

	// Add message to context
	return contextManager.AddMessage(ctx, sessionID, message)
}

// processAgentEvents is no longer needed - events are handled directly by Handler layer via EventBus subscription
