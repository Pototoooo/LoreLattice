package chatpipline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

// PluginStreamFilter implements stream filtering functionality for chat pipeline
type PluginStreamFilter struct{}

// NewPluginStreamFilter creates a new stream filter plugin instance
func NewPluginStreamFilter(eventManager *EventManager) *PluginStreamFilter {
	res := &PluginStreamFilter{}
	eventManager.Register(res)
	return res
}

// ActivationEvents returns the event types this plugin handles
func (p *PluginStreamFilter) ActivationEvents() []types.EventType {
	return []types.EventType{types.STREAM_FILTER}
}

// OnEvent handles stream filtering events in the chat pipeline
func (p *PluginStreamFilter) OnEvent(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage, next func() *PluginError,
) *PluginError {
	logger.Info(ctx, "Starting stream filter")

	// EventBus is required
	if chatManage.EventBus == nil {
		logger.Error(ctx, "EventBus is required but not available")
		return ErrModelCall.WithError(errors.New("EventBus is required for stream filtering"))
	}
	eventBus := chatManage.EventBus

	// Check if no-match prefix filtering is needed
	matchNoMatchBuilderPrefix := chatManage.SummaryConfig.NoMatchPrefix != ""

	if matchNoMatchBuilderPrefix {
		logger.Infof(ctx, "Using no match prefix filter: %s", chatManage.SummaryConfig.NoMatchPrefix)
		// Create an event interceptor for prefix filtering
		return p.filterEventsWithPrefix(ctx, chatManage, eventBus, next)
	}

	// No filtering needed, just pass through
	logger.Info(ctx, "No prefix filtering required, passing through")
	return next()
}

// filterEventsWithPrefix intercepts events, checks for NoMatchPrefix, and re-emits filtered events
func (p *PluginStreamFilter) filterEventsWithPrefix(
	ctx context.Context,
	chatManage *types.ChatManage,
	originalEventBus types.EventBusInterface,
	next func() *PluginError,
) *PluginError {
	logger.Info(ctx, "Setting up event-based stream filtering with NoMatchPrefix")

	// Create a temporary EventBus to intercept events
	tempEventBus := event.NewEventBus()
	chatManage.EventBus = tempEventBus.AsEventBusInterface()

	responseBuilder := &strings.Builder{}
	matchFound := false

	// Subscribe to answer events from temp bus
	tempEventBus.On(event.EventAgentFinalAnswer, func(ctx context.Context, evt event.Event) error {
		data, ok := evt.Data.(event.AgentFinalAnswerData)
		if !ok {
			return nil
		}

		responseBuilder.WriteString(data.Content)

		// Check if content does NOT match the no-match prefix (meaning it's valid content)
		if !strings.HasPrefix(chatManage.SummaryConfig.NoMatchPrefix, responseBuilder.String()) {
			logger.Infof(ctx, "Content does not match no-match prefix, emitting valid content: %s", responseBuilder.String())

			// Emit the accumulated content as valid answer
			originalEventBus.Emit(ctx, types.Event{
				ID:        evt.ID,
				Type:      types.EventType(event.EventAgentFinalAnswer),
				SessionID: chatManage.SessionID,
				Data: event.AgentFinalAnswerData{
					Content: responseBuilder.String(),
					Done:    data.Done,
				},
			})
			matchFound = true
		}

		return nil
	})

	// Call next to trigger pipeline stages that will emit to tempEventBus
	err := next()

	// After pipeline completes, check if we need fallback
	if !matchFound && responseBuilder.Len() > 0 {
		logger.Info(ctx, "Content matches no-match prefix, emitting fallback response")
		fallbackID := fmt.Sprintf("%s-fallback", uuid.New().String()[:8])
		originalEventBus.Emit(ctx, types.Event{
			ID:        fallbackID,
			Type:      types.EventType(event.EventAgentFinalAnswer),
			SessionID: chatManage.SessionID,
			Data: event.AgentFinalAnswerData{
				Content: chatManage.FallbackResponse,
				Done:    true,
			},
		})
	}

	// Restore original EventBus
	chatManage.EventBus = originalEventBus

	return err
}
