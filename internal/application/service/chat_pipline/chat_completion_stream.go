package chatpipline

import (
	"context"
	"errors"
	"fmt"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

// PluginChatCompletionStream implements streaming chat completion functionality
// as a plugin that can be registered to EventManager
type PluginChatCompletionStream struct {
	modelService interfaces.ModelService // Interface for model operations
}

// NewPluginChatCompletionStream creates a new PluginChatCompletionStream instance
// and registers it with the EventManager
func NewPluginChatCompletionStream(eventManager *EventManager,
	modelService interfaces.ModelService,
) *PluginChatCompletionStream {
	res := &PluginChatCompletionStream{
		modelService: modelService,
	}
	eventManager.Register(res)
	return res
}

// ActivationEvents returns the event types this plugin handles
func (p *PluginChatCompletionStream) ActivationEvents() []types.EventType {
	return []types.EventType{types.CHAT_COMPLETION_STREAM}
}

// OnEvent handles streaming chat completion events
// It prepares the chat model, messages, and initiates streaming response
func (p *PluginChatCompletionStream) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	logger.Info(ctx, "Starting chat completion stream")

	// Prepare chat model and options
	chatModel, opt, err := prepareChatModel(ctx, p.modelService, chatManage)
	if err != nil {
		return ErrGetChatModel.WithError(err)
	}

	// Prepare base messages without history
	logger.Info(ctx, "Preparing chat messages")
	chatMessages := prepareMessagesWithHistory(chatManage)

	// EventBus is required for event-driven streaming
	if chatManage.EventBus == nil {
		logger.Error(ctx, "EventBus is required but not available")
		return ErrModelCall.WithError(errors.New("EventBus is required for streaming"))
	}
	eventBus := chatManage.EventBus

	logger.Info(ctx, "EventBus detected, enabling event-driven streaming mode")

	// Initiate streaming chat model call with independent context
	logger.Info(ctx, "Calling chat stream model")
	responseChan, err := chatModel.ChatStream(ctx, chatMessages, opt)
	if err != nil {
		logger.Errorf(ctx, "Failed to call chat stream model: %v", err)
		return ErrModelCall.WithError(err)
	}
	if responseChan == nil {
		logger.Error(ctx, "Chat stream returned nil channel")
		return ErrModelCall.WithError(errors.New("chat stream returned nil channel"))
	}

	logger.Info(ctx, "Chat stream initiated successfully")

	// Start goroutine to consume channel and emit events directly
	go func() {
		answerID := fmt.Sprintf("%s-answer", uuid.New().String()[:8])
		var finalContent string

		for response := range responseChan {
			// Emit event for each answer chunk
			if response.ResponseType == types.ResponseTypeAnswer {
				finalContent += response.Content
				if err := eventBus.Emit(ctx, types.Event{
					ID:        answerID,
					Type:      types.EventType(event.EventAgentFinalAnswer),
					SessionID: chatManage.SessionID,
					Data: event.AgentFinalAnswerData{
						Content: response.Content,
						Done:    response.Done,
					},
				}); err != nil {
					logger.Errorf(ctx, "Failed to emit answer event: %v", err)
				}
			}
		}

		logger.Info(ctx, "Chat stream completed, emitting completion event")

		// Emit completion event when stream finishes
		// This allows other components to detect stream completion
		if err := eventBus.Emit(ctx, types.Event{
			ID:        fmt.Sprintf("%s-complete", uuid.New().String()[:8]),
			Type:      types.EventType(event.EventAgentComplete),
			SessionID: chatManage.SessionID,
			Data: event.AgentCompleteData{
				SessionID:   chatManage.SessionID,
				MessageID:   chatManage.MessageID,
				FinalAnswer: finalContent,
			},
		}); err != nil {
			logger.Errorf(ctx, "Failed to emit completion event: %v", err)
		}

		logger.Info(ctx, "Chat stream completed and completion event emitted")
	}()

	return next()
}
