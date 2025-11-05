package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// sessionContext holds the context for a single session
type sessionContext struct {
	Messages []chat.Message
	mu       sync.RWMutex
}

// contextManager implements the ContextManager interface
type contextManager struct {
	sessions            map[string]*sessionContext
	mu                  sync.RWMutex
	compressionStrategy interfaces.CompressionStrategy
	maxTokens           int // Maximum tokens allowed in context
}

// NewContextManager creates a new context manager
func NewContextManager(compressionStrategy interfaces.CompressionStrategy, maxTokens int) interfaces.ContextManager {
	return &contextManager{
		sessions:            make(map[string]*sessionContext),
		compressionStrategy: compressionStrategy,
		maxTokens:           maxTokens,
	}
}

// getOrCreateSession gets or creates a session context
func (cm *contextManager) getOrCreateSession(sessionID string) *sessionContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if session, exists := cm.sessions[sessionID]; exists {
		return session
	}

	session := &sessionContext{
		Messages: make([]chat.Message, 0),
	}
	cm.sessions[sessionID] = session
	return session
}

// AddMessage adds a message to the session context
func (cm *contextManager) AddMessage(ctx context.Context, sessionID string, message chat.Message) error {
	logger.Infof(ctx, "[ContextManager][Session-%s] Adding message: role=%s, content_length=%d",
		sessionID, message.Role, len(message.Content))
	logger.Debugf(ctx, "Adding message to context, session: %s, role: %s", sessionID, message.Role)

	// Log message content preview
	contentPreview := message.Content
	if len(contentPreview) > 200 {
		contentPreview = contentPreview[:200] + "..."
	}
	logger.Debugf(ctx, "[ContextManager][Session-%s] Message content preview: %s", sessionID, contentPreview)

	session := cm.getOrCreateSession(sessionID)
	session.mu.Lock()
	defer session.mu.Unlock()

	beforeCount := len(session.Messages)
	session.Messages = append(session.Messages, message)
	logger.Debugf(ctx, "[ContextManager][Session-%s] Messages count: %d -> %d", sessionID, beforeCount, len(session.Messages))

	// Check if compression is needed
	tokenCount := cm.compressionStrategy.EstimateTokens(session.Messages)
	logger.Debugf(ctx, "[ContextManager][Session-%s] Current token count: %d (max: %d)",
		sessionID, tokenCount, cm.maxTokens)

	if tokenCount > cm.maxTokens {
		logger.Infof(ctx, "[ContextManager][Session-%s] Context exceeds max tokens (%d > %d), applying compression",
			sessionID, tokenCount, cm.maxTokens)
		beforeCompressionCount := len(session.Messages)
		compressed, err := cm.compressionStrategy.Compress(ctx, session.Messages, cm.maxTokens)
		if err != nil {
			logger.Errorf(ctx, "[ContextManager][Session-%s] Failed to compress context: %v", sessionID, err)
			return fmt.Errorf("failed to compress context: %w", err)
		}
		session.Messages = compressed
		afterTokenCount := cm.compressionStrategy.EstimateTokens(session.Messages)
		logger.Infof(ctx, "[ContextManager][Session-%s] Context compressed: %d -> %d messages, %d -> %d tokens",
			sessionID, beforeCompressionCount, len(compressed), tokenCount, afterTokenCount)
	}

	return nil
}

// GetContext retrieves the current context for a session
func (cm *contextManager) GetContext(ctx context.Context, sessionID string) ([]chat.Message, error) {
	logger.Infof(ctx, "[ContextManager][Session-%s] Getting context", sessionID)
	logger.Debugf(ctx, "Getting context for session: %s", sessionID)

	session := cm.getOrCreateSession(sessionID)
	session.mu.RLock()
	defer session.mu.RUnlock()

	// Return a copy to avoid external modifications
	messages := make([]chat.Message, len(session.Messages))
	copy(messages, session.Messages)

	// Calculate token estimate
	tokenCount := cm.compressionStrategy.EstimateTokens(messages)

	logger.Infof(ctx, "[ContextManager][Session-%s] Retrieved %d messages (~%d tokens)",
		sessionID, len(messages), tokenCount)
	logger.Debugf(ctx, "Retrieved %d messages from context", len(messages))

	// Log message role distribution
	roleCount := make(map[string]int)
	for _, msg := range messages {
		roleCount[msg.Role]++
	}
	logger.Debugf(ctx, "[ContextManager][Session-%s] Message distribution: %v", sessionID, roleCount)

	return messages, nil
}

// ClearContext clears all context for a session
func (cm *contextManager) ClearContext(ctx context.Context, sessionID string) error {
	logger.Infof(ctx, "[ContextManager][Session-%s] Clearing context", sessionID)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if session exists before deleting
	if session, exists := cm.sessions[sessionID]; exists {
		messageCount := len(session.Messages)
		logger.Infof(ctx, "[ContextManager][Session-%s] Deleted %d messages", sessionID, messageCount)
	} else {
		logger.Debugf(ctx, "[ContextManager][Session-%s] Session not found (already cleared or never created)",
			sessionID)
	}

	delete(cm.sessions, sessionID)
	return nil
}

// GetContextStats returns statistics about the context
func (cm *contextManager) GetContextStats(ctx context.Context, sessionID string) (*interfaces.ContextStats, error) {
	session := cm.getOrCreateSession(sessionID)
	session.mu.RLock()
	defer session.mu.RUnlock()

	tokenCount := cm.compressionStrategy.EstimateTokens(session.Messages)

	stats := &interfaces.ContextStats{
		MessageCount:         len(session.Messages),
		TokenCount:           tokenCount,
		IsCompressed:         false, // We'd need to track this explicitly for accurate reporting
		OriginalMessageCount: len(session.Messages),
	}

	logger.Debugf(ctx, "Context stats for session %s: %d messages, ~%d tokens",
		sessionID, stats.MessageCount, stats.TokenCount)

	return stats, nil
}

// SlidingWindowStrategy implements a sliding window compression strategy
// It keeps the most recent N messages and discards older ones
type SlidingWindowStrategy struct {
	windowSize int // Number of messages to keep
}

// NewSlidingWindowStrategy creates a new sliding window strategy
func NewSlidingWindowStrategy(windowSize int) interfaces.CompressionStrategy {
	return &SlidingWindowStrategy{
		windowSize: windowSize,
	}
}

// Compress applies sliding window compression
func (s *SlidingWindowStrategy) Compress(ctx context.Context, messages []chat.Message, maxTokens int) ([]chat.Message, error) {
	if len(messages) <= s.windowSize {
		return messages, nil
	}

	// Keep system messages and recent messages
	var compressed []chat.Message
	var systemMessages []chat.Message
	var otherMessages []chat.Message

	// Separate system messages
	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// Keep last N non-system messages
	startIdx := 0
	if len(otherMessages) > s.windowSize {
		startIdx = len(otherMessages) - s.windowSize
	}

	// Combine system messages with recent messages
	compressed = append(compressed, systemMessages...)
	compressed = append(compressed, otherMessages[startIdx:]...)

	logger.Infof(ctx, "Sliding window compression: %d -> %d messages (kept %d system, %d recent)",
		len(messages), len(compressed), len(systemMessages), len(compressed)-len(systemMessages))

	return compressed, nil
}

// EstimateTokens provides a rough token count estimation
// Uses approximately 4 characters per token as a heuristic
func (s *SlidingWindowStrategy) EstimateTokens(messages []chat.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	// Rough estimate: ~4 characters per token
	return totalChars / 4
}

// SmartCompressionStrategy implements intelligent context compression
// It uses summarization for older messages while keeping recent ones intact
type SmartCompressionStrategy struct {
	recentMessageCount int       // Number of recent messages to keep intact
	chatModel          chat.Chat // LLM for summarization
	summarizeThreshold int       // Minimum messages before summarization
}

// NewSmartCompressionStrategy creates a new smart compression strategy
func NewSmartCompressionStrategy(recentMessageCount int, chatModel chat.Chat, summarizeThreshold int) interfaces.CompressionStrategy {
	return &SmartCompressionStrategy{
		recentMessageCount: recentMessageCount,
		chatModel:          chatModel,
		summarizeThreshold: summarizeThreshold,
	}
}

// Compress applies smart compression with summarization
func (s *SmartCompressionStrategy) Compress(ctx context.Context, messages []chat.Message, maxTokens int) ([]chat.Message, error) {
	logger.Infof(ctx, "[ContextManager][Compress] Smart compression started: %d messages, max_tokens=%d",
		len(messages), maxTokens)

	if len(messages) <= s.recentMessageCount {
		logger.Infof(ctx, "[ContextManager][Compress] No compression needed (%d <= %d)",
			len(messages), s.recentMessageCount)
		return messages, nil
	}

	// Separate messages into system, old, and recent
	var systemMessages []chat.Message
	var oldMessages []chat.Message
	var recentMessages []chat.Message

	for i, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else if i < len(messages)-s.recentMessageCount {
			oldMessages = append(oldMessages, msg)
		} else {
			recentMessages = append(recentMessages, msg)
		}
	}

	logger.Infof(ctx, "[ContextManager][Compress] Message distribution: system=%d, old=%d, recent=%d",
		len(systemMessages), len(oldMessages), len(recentMessages))

	// If we have enough old messages, summarize them
	var compressed []chat.Message
	if len(oldMessages) >= s.summarizeThreshold && s.chatModel != nil {
		logger.Infof(ctx, "[ContextManager][Compress] Summarizing %d old messages (threshold: %d)",
			len(oldMessages), s.summarizeThreshold)

		// Build conversation text for summarization
		var conversationText strings.Builder
		for _, msg := range oldMessages {
			conversationText.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}

		// Create summarization prompt
		summaryPrompt := []chat.Message{
			{
				Role: "system",
				Content: "You are a helpful assistant that summarizes conversation history. " +
					"Summarize the following conversation concisely while preserving key information and context. " +
					"Keep the summary under 200 words.",
			},
			{
				Role:    "user",
				Content: conversationText.String(),
			},
		}

		logger.Debugf(ctx, "[ContextManager][Compress] Conversation text length: %d chars",
			conversationText.Len())

		// Get summary from LLM
		logger.Infof(ctx, "[ContextManager][Compress] Calling LLM for summarization...")
		response, err := s.chatModel.Chat(ctx, summaryPrompt, &chat.ChatOptions{
			Temperature: 0.3,
		})
		if err != nil {
			logger.Warnf(ctx, "[ContextManager][Compress] Failed to summarize old messages: %v, falling back to truncation",
				err)
			// Fallback to keeping only recent messages
		} else {
			// Add summary as a system message
			summaryContent := fmt.Sprintf("[Previous conversation summary]: %s", response.Content)
			compressed = append(compressed, chat.Message{
				Role:    "system",
				Content: summaryContent,
			})
			logger.Infof(ctx, "[ContextManager][Compress] Successfully summarized %d messages into %d chars",
				len(oldMessages), len(summaryContent))
			logger.Debugf(ctx, "[ContextManager][Compress] Summary: %s", response.Content)
		}
	} else {
		// Not enough messages to summarize, keep them as is
		logger.Infof(ctx, "[ContextManager][Compress] Not enough messages to summarize (%d < %d), keeping as is",
			len(oldMessages), s.summarizeThreshold)
		compressed = append(compressed, oldMessages...)
	}

	// Add system messages, compressed history, and recent messages
	result := make([]chat.Message, 0, len(systemMessages)+len(compressed)+len(recentMessages))
	result = append(result, systemMessages...)
	result = append(result, compressed...)
	result = append(result, recentMessages...)

	beforeTokens := s.EstimateTokens(messages)
	afterTokens := s.EstimateTokens(result)

	logger.Infof(ctx, "[ContextManager][Compress] Smart compression completed: %d -> %d messages (system: %d, compressed: %d, recent: %d), tokens: %d -> %d",
		len(messages), len(result), len(systemMessages), len(compressed), len(recentMessages),
		beforeTokens, afterTokens)

	return result, nil
}

// EstimateTokens provides a rough token count estimation
func (s *SmartCompressionStrategy) EstimateTokens(messages []chat.Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	// Rough estimate: ~4 characters per token
	return totalChars / 4
}
