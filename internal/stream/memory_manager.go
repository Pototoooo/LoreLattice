package stream

import (
	"context"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// 内存流信息
type memoryStreamInfo struct {
	sessionID           string
	requestID           string
	query               string
	events              []interfaces.StreamEvent
	knowledgeReferences types.References
	lastUpdated         time.Time
	isCompleted         bool
}

// MemoryStreamManager 基于内存的流管理器实现
type MemoryStreamManager struct {
	// 会话ID -> 请求ID -> 流数据
	activeStreams map[string]map[string]*memoryStreamInfo
	mu            sync.RWMutex
}

// NewMemoryStreamManager 创建一个新的内存流管理器
func NewMemoryStreamManager() *MemoryStreamManager {
	return &MemoryStreamManager{
		activeStreams: make(map[string]map[string]*memoryStreamInfo),
	}
}

// RegisterStream 注册一个新的流
func (m *MemoryStreamManager) RegisterStream(ctx context.Context, sessionID, requestID, query string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := &memoryStreamInfo{
		sessionID:   sessionID,
		requestID:   requestID,
		query:       query,
		lastUpdated: time.Now(),
	}

	if _, exists := m.activeStreams[sessionID]; !exists {
		m.activeStreams[sessionID] = make(map[string]*memoryStreamInfo)
	}

	m.activeStreams[sessionID][requestID] = info
	return nil
}

// PushEvent 推送事件到流（追加模式）
func (m *MemoryStreamManager) PushEvent(ctx context.Context,
	sessionID, requestID string, event interfaces.StreamEvent,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionMap, exists := m.activeStreams[sessionID]; exists {
		if stream, found := sessionMap[requestID]; found {
			stream.events = append(stream.events, event)
			stream.lastUpdated = time.Now()
		}
	}
	return nil
}

// ReplaceEvent 通过 ID 替换事件（用于流式进度更新）
func (m *MemoryStreamManager) ReplaceEvent(ctx context.Context,
	sessionID, requestID string, event interfaces.StreamEvent,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionMap, exists := m.activeStreams[sessionID]; exists {
		if stream, found := sessionMap[requestID]; found {
			// 通过 ID 精确查找并替换
			replaced := false
			for i := range stream.events {
				if stream.events[i].ID == event.ID {
					// 找到了，替换
					stream.events[i] = event
					replaced = true
					break
				}
			}
			// 没找到，追加
			if !replaced {
				stream.events = append(stream.events, event)
			}
			stream.lastUpdated = time.Now()
		}
	}
	return nil
}

// UpdateReferences 更新知识引用
func (m *MemoryStreamManager) UpdateReferences(ctx context.Context,
	sessionID, requestID string, references types.References,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionMap, exists := m.activeStreams[sessionID]; exists {
		if stream, found := sessionMap[requestID]; found {
			stream.knowledgeReferences = references
			stream.lastUpdated = time.Now()
		}
	}
	return nil
}

// CompleteStream 完成流
func (m *MemoryStreamManager) CompleteStream(ctx context.Context, sessionID, requestID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionMap, exists := m.activeStreams[sessionID]; exists {
		if stream, found := sessionMap[requestID]; found {
			stream.isCompleted = true
			// 30s 后删除流
			go func() {
				time.Sleep(30 * time.Second)
				m.mu.Lock()
				defer m.mu.Unlock()
				delete(sessionMap, requestID)
				if len(sessionMap) == 0 {
					delete(m.activeStreams, sessionID)
				}
			}()
		}
	}
	return nil
}

// GetStream 获取特定流
func (m *MemoryStreamManager) GetStream(ctx context.Context,
	sessionID, requestID string,
) (*interfaces.StreamInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionMap, exists := m.activeStreams[sessionID]; exists {
		if stream, found := sessionMap[requestID]; found {
			return &interfaces.StreamInfo{
				SessionID:           stream.sessionID,
				RequestID:           stream.requestID,
				Query:               stream.query,
				Events:              stream.events,
				KnowledgeReferences: stream.knowledgeReferences,
				LastUpdated:         stream.lastUpdated,
				IsCompleted:         stream.isCompleted,
			}, nil
		}
	}
	return nil, nil
}

// 确保实现了接口
var _ interfaces.StreamManager = (*MemoryStreamManager)(nil)
