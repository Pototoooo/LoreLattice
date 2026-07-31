package chat

import (
	"context"
	"testing"
	"time"

	"github.com/Pototoooo/lorelattice/internal/billing"
	"github.com/Pototoooo/lorelattice/internal/types"
)

type meteringFakeChat struct {
	usage           types.TokenUsage
	streamResponses []types.StreamResponse
	chatCalls       int
}

func (f *meteringFakeChat) GetModelName() string { return "test-model" }
func (f *meteringFakeChat) GetModelID() string   { return "model-1" }
func (f *meteringFakeChat) Chat(context.Context, []Message, *ChatOptions) (*types.ChatResponse, error) {
	f.chatCalls++
	return &types.ChatResponse{Usage: f.usage}, nil
}
func (f *meteringFakeChat) ChatStream(context.Context, []Message, *ChatOptions) (<-chan types.StreamResponse, error) {
	f.chatCalls++
	responses := f.streamResponses
	if responses == nil {
		responses = []types.StreamResponse{{Content: "ok"}, {Done: true, Usage: &f.usage}}
	}
	ch := make(chan types.StreamResponse, len(responses))
	for _, response := range responses {
		ch <- response
	}
	close(ch)
	return ch, nil
}

type fakeBillingRuntime struct {
	reservedTenant   uint64
	reservedQuantity float64
	completed        chan float64
	estimated        chan bool
	reserveErr       error
}

func (f *fakeBillingRuntime) Enabled() bool                 { return true }
func (f *fakeBillingRuntime) DefaultCompletionReserve() int { return 32 }
func (f *fakeBillingRuntime) Reserve(_ context.Context, tenantID uint64, _ billing.FeatureKey, quantity float64, _ billing.UsageMetadata) (*billing.Reservation, error) {
	f.reservedTenant, f.reservedQuantity = tenantID, quantity
	if f.reserveErr != nil {
		return nil, f.reserveErr
	}
	return &billing.Reservation{ID: "reservation-1", TenantID: tenantID, UnitPrice: 0.0001}, nil
}
func (f *fakeBillingRuntime) Complete(_ context.Context, _ *billing.Reservation, actual float64, estimated bool) error {
	f.completed <- actual
	f.estimated <- estimated
	return nil
}
func (f *fakeBillingRuntime) Release(context.Context, *billing.Reservation, string) error { return nil }

func meteringContext() context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(200))
	ctx = context.WithValue(ctx, types.SessionTenantIDContextKey, uint64(100))
	return context.WithValue(ctx, types.RequestIDContextKey, "req-1")
}

func newFakeRuntime() *fakeBillingRuntime {
	return &fakeBillingRuntime{completed: make(chan float64, 1), estimated: make(chan bool, 1)}
}

func TestMeterForgeChatReservesAndCompletesNonStreamingUsage(t *testing.T) {
	runtime := newFakeRuntime()
	wrapper := &meterForgeChat{
		inner: &meteringFakeChat{usage: types.TokenUsage{
			PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15,
		}},
		service: runtime, provider: "test-provider",
	}
	if _, err := wrapper.Chat(meteringContext(), nil, nil); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if runtime.reservedTenant != 100 {
		t.Fatalf("tenant = %d; want session tenant 100", runtime.reservedTenant)
	}
	if got := <-runtime.completed; got != 15 {
		t.Fatalf("completed quantity = %v", got)
	}
	if got := <-runtime.estimated; got {
		t.Fatal("provider usage must not be marked estimated")
	}
}

func TestMeterForgeChatReportsStreamingUsageOnce(t *testing.T) {
	runtime := newFakeRuntime()
	wrapper := &meterForgeChat{
		inner: &meteringFakeChat{usage: types.TokenUsage{
			PromptTokens: 8, CompletionTokens: 2, TotalTokens: 10,
		}},
		service: runtime, provider: "test-provider",
	}
	stream, err := wrapper.ChatStream(meteringContext(), nil, nil)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range stream {
	}
	if got := <-runtime.completed; got != 10 {
		t.Fatalf("completed quantity = %v", got)
	}
}

func TestMeterForgeChatEstimatesStreamingUsageWhenProviderOmitsUsage(t *testing.T) {
	runtime := newFakeRuntime()
	wrapper := &meterForgeChat{
		inner: &meteringFakeChat{streamResponses: []types.StreamResponse{
			{Content: "MeterForge "}, {Content: "integration complete", Done: true},
		}},
		service: runtime, provider: "openai-compatible",
	}
	stream, err := wrapper.ChatStream(meteringContext(), []Message{
		{Role: "system", Content: "Reply briefly."},
		{Role: "user", Content: "Confirm integration."},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range stream {
	}
	if got := <-runtime.completed; got <= 0 {
		t.Fatalf("estimated quantity = %v", got)
	}
	if got := <-runtime.estimated; !got {
		t.Fatal("expected estimated usage")
	}
}

func TestMeterForgeChatDrainsUsageChunkAfterConsumerStopsAtDone(t *testing.T) {
	runtime := newFakeRuntime()
	usage := types.TokenUsage{PromptTokens: 20, CompletionTokens: 4, TotalTokens: 24}
	wrapper := &meterForgeChat{
		inner: &meteringFakeChat{streamResponses: []types.StreamResponse{
			{Content: "done"}, {Done: true, FinishReason: "stop"}, {Done: true, Usage: &usage},
		}},
		service: runtime, provider: "openai-compatible",
	}
	stream, err := wrapper.ChatStream(meteringContext(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for response := range stream {
		if response.Done {
			break
		}
	}
	select {
	case got := <-runtime.completed:
		if got != 24 {
			t.Fatalf("completed quantity = %v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("usage was not settled after downstream stopped at Done")
	}
}

func TestMeterForgeChatBlocksProviderWhenReservationFails(t *testing.T) {
	runtime := newFakeRuntime()
	runtime.reserveErr = &billing.BillingError{
		Code: "insufficient_credit", Message: "余额不足", HTTPStatus: 402,
		Feature: billing.FeatureLLMTokens,
	}
	inner := &meteringFakeChat{}
	wrapper := &meterForgeChat{inner: inner, service: runtime}
	if _, err := wrapper.Chat(meteringContext(), nil, nil); err == nil {
		t.Fatal("expected billing error")
	}
	if inner.chatCalls != 0 {
		t.Fatalf("provider calls = %d; want 0", inner.chatCalls)
	}
}
