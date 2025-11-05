package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

// StreamEvent represents a single event in the stream
type StreamEvent struct {
	ID        string                 `json:"id"`             // Unique event ID for precise replacement
	Type      types.ResponseType     `json:"type"`           // Event type (thinking, tool_call, tool_result, etc.)
	Content   string                 `json:"content"`        // Event content
	Done      bool                   `json:"done"`           // Whether this event is done
	Timestamp time.Time              `json:"timestamp"`      // When this event occurred
	Data      map[string]interface{} `json:"data,omitempty"` // Additional event data
}

// StreamInfo stream information
type StreamInfo struct {
	SessionID           string           // session ID
	RequestID           string           // request ID
	Query               string           // query content
	Events              []StreamEvent    // all events in order (for replay on refresh)
	KnowledgeReferences types.References // knowledge references
	LastUpdated         time.Time        // last updated time
	IsCompleted         bool             // whether completed
}

// StreamManager stream manager interface
type StreamManager interface {
	// RegisterStream registers a new stream
	RegisterStream(ctx context.Context, sessionID, requestID, query string) error

	// PushEvent pushes an event to the stream (append mode)
	PushEvent(ctx context.Context, sessionID, requestID string, event StreamEvent) error

	// ReplaceEvent replaces an event by ID (for streaming progress updates)
	// If no matching event exists, it will append the event
	ReplaceEvent(ctx context.Context, sessionID, requestID string, event StreamEvent) error

	// UpdateReferences updates knowledge references
	UpdateReferences(ctx context.Context, sessionID, requestID string, references types.References) error

	// CompleteStream completes the stream
	CompleteStream(ctx context.Context, sessionID, requestID string) error

	// GetStream gets a specific stream (for replay on refresh)
	GetStream(ctx context.Context, sessionID, requestID string) (*StreamInfo, error)
}
