package chat

import (
	"context"
	"github.com/Pototoooo/lorelattice/internal/logger"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/tiktoken-go/tokenizer"
)

const (
	meteringMessageOverhead  = 3
	meteringConversationTail = 3
)

var (
	meteringTokenizerOnce sync.Once
	meteringTokenizer     tokenizer.Codec
)

type billingRuntime interface {
	Enabled() bool
	DefaultCompletionReserve() int
	Reserve(context.Context, uint64, billing.FeatureKey, float64, billing.UsageMetadata) (*billing.Reservation, error)
	Complete(context.Context, *billing.Reservation, float64, bool) error
	Release(context.Context, *billing.Reservation, string) error
}

// meterForgeChat reserves quota and prepaid credit before every real provider
// call, then records exactly one durable event after the provider finishes.
type meterForgeChat struct {
	inner    Chat
	service  billingRuntime
	provider string
	mode     billing.BillingMode
}

func (w *meterForgeChat) GetModelName() string { return w.inner.GetModelName() }
func (w *meterForgeChat) GetModelID() string   { return w.inner.GetModelID() }

func (w *meterForgeChat) Chat(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (*types.ChatResponse, error) {
	reservation, callOpts, err := w.reserve(ctx, messages, opts, "chat")
	if err != nil {
		return nil, err
	}
	response, err := w.inner.Chat(ctx, messages, callOpts)
	if err != nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, err.Error())
		return response, err
	}
	if response == nil {
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, "empty provider response")
		return nil, nil
	}
	usage, estimated := usageOrEstimate(response.Usage, messages, response.Content, response.ToolCalls)
	if err := w.service.Complete(context.WithoutCancel(ctx), reservation, float64(usage.TotalTokens), estimated); err != nil {
		return nil, err
	}
	return response, nil
}

func (w *meterForgeChat) ChatStream(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (<-chan types.StreamResponse, error) {
	reservation, callOpts, err := w.reserve(ctx, messages, opts, "chat_stream")
	if err != nil {
		return nil, err
	}
	stream, err := w.inner.ChatStream(ctx, messages, callOpts)
	if err != nil || stream == nil {
		reason := "empty provider stream"
		if err != nil {
			reason = err.Error()
		}
		_ = w.service.Release(context.WithoutCancel(ctx), reservation, reason)
		return stream, err
	}

	out := make(chan types.StreamResponse)
	go func() {
		defer close(out)
		var usage *types.TokenUsage
		var content string
		var toolCalls []types.LLMToolCall
		downstreamDone := false
		for response := range stream {
			if response.Usage != nil {
				copied := *response.Usage
				usage = &copied
			}
			content += response.Content
			if len(response.ToolCalls) > 0 {
				toolCalls = response.ToolCalls
			}
			if downstreamDone {
				continue
			}
			select {
			case out <- response:
				if response.Done {
					downstreamDone = true
				}
			case <-ctx.Done():
				// Drain usage-only chunks without blocking the completed SSE
				// consumer, then settle the already-performed provider call.
				for response := range stream {
					if response.Usage != nil {
						copied := *response.Usage
						usage = &copied
					}
					content += response.Content
					if len(response.ToolCalls) > 0 {
						toolCalls = response.ToolCalls
					}
				}
				resolved, estimated := usageOrEstimatePtr(usage, messages, content, toolCalls)
				w.completeStream(ctx, reservation, float64(resolved.TotalTokens), estimated)
				return
			}
		}
		resolved, estimated := usageOrEstimatePtr(usage, messages, content, toolCalls)
		w.completeStream(ctx, reservation, float64(resolved.TotalTokens), estimated)
	}()
	return out, nil
}

func (w *meterForgeChat) reserve(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
	operation string,
) (*billing.Reservation, *ChatOptions, error) {
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, nil, billing.ToAppError(&billing.BillingError{
			Code: "billing_not_ready", Message: "模型调用缺少 workspace 计费上下文", HTTPStatus: 503,
			Feature: billing.FeatureLLMTokens,
		})
	}
	callOpts := cloneChatOptions(opts)
	outputReserve := callOpts.MaxCompletionTokens
	if outputReserve <= 0 {
		outputReserve = callOpts.MaxTokens
	}
	if outputReserve <= 0 {
		outputReserve = w.service.DefaultCompletionReserve()
		if w.mode.Chargeable() || w.mode == billing.BillingModeIncluded || !w.mode.Valid() {
			callOpts.MaxTokens = outputReserve
		}
	}
	quantity := estimateMessages(messages) + outputReserve
	requestID, _ := types.RequestIDFromContext(ctx)
	reservation, err := w.service.Reserve(ctx, tenantID, billing.FeatureLLMTokens, float64(quantity), billing.UsageMetadata{
		ModelID: w.GetModelID(), ModelName: w.GetModelName(), Provider: w.provider,
		Operation: operation, RequestID: requestID, JobID: requestID,
		Category: "chat_agent", Mode: w.mode, Estimated: true,
	})
	return reservation, callOpts, billing.ToAppError(err)
}

func cloneChatOptions(opts *ChatOptions) *ChatOptions {
	if opts == nil {
		return &ChatOptions{}
	}
	copy := *opts
	return &copy
}

func usageOrEstimate(
	usage types.TokenUsage,
	messages []Message,
	content string,
	toolCalls []types.LLMToolCall,
) (types.TokenUsage, bool) {
	return usageOrEstimatePtr(&usage, messages, content, toolCalls)
}

func usageOrEstimatePtr(
	usage *types.TokenUsage,
	messages []Message,
	content string,
	toolCalls []types.LLMToolCall,
) (types.TokenUsage, bool) {
	if usage != nil {
		resolved := *usage
		if resolved.TotalTokens <= 0 {
			resolved.TotalTokens = resolved.PromptTokens + resolved.CompletionTokens
		}
		if resolved.TotalTokens > 0 {
			return resolved, false
		}
	}
	promptTokens := estimateMessages(messages)
	completionTokens := estimateText(content)
	for _, call := range toolCalls {
		completionTokens += estimateText(call.Function.Name)
		completionTokens += estimateText(call.Function.Arguments)
		completionTokens += 4
	}
	return types.TokenUsage{
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
		TotalTokens: promptTokens + completionTokens,
	}, true
}

func estimateMessages(messages []Message) int {
	total := meteringConversationTail
	for _, message := range messages {
		total += meteringMessageOverhead
		total += estimateText(message.Role)
		total += estimateText(message.Content)
		total += estimateText(message.Name)
		for _, call := range message.ToolCalls {
			total += estimateText(call.Function.Name)
			total += estimateText(call.Function.Arguments)
			total += 4
		}
	}
	return total
}

func estimateText(value string) int {
	if value == "" {
		return 0
	}
	meteringTokenizerOnce.Do(func() {
		codec, err := tokenizer.Get(tokenizer.Cl100kBase)
		if err == nil {
			meteringTokenizer = codec
		}
	})
	if meteringTokenizer != nil {
		ids, _, err := meteringTokenizer.Encode(value)
		if err == nil {
			return len(ids)
		}
	}
	runes := utf8.RuneCountInString(value)
	asciiApprox := (len(value) + 3) / 4
	if runes > asciiApprox {
		return runes
	}
	return asciiApprox
}

func wrapChatMeterForge(c Chat, config *ChatConfig, err error) (Chat, error) {
	if err != nil || c == nil {
		return c, err
	}
	service := billing.Default()
	if service == nil || !service.Enabled() {
		return c, nil
	}
	mode := billing.ResolveBillingMode(config.Source, config.APIKey, config.Provider, config.ExtraConfig)
	return &meterForgeChat{inner: c, service: service, provider: config.Provider, mode: mode}, nil
}

func (w *meterForgeChat) completeStream(ctx context.Context, reservation *billing.Reservation, actual float64, estimated bool) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = w.service.Complete(ctx, reservation, actual, estimated); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			attempt = 3
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
	id := ""
	if reservation != nil {
		id = reservation.ID
	}
	logger.Errorf(ctx, "[Billing] stream settlement %s failed; durable reservation retained for recovery: %v", id, err)
}
