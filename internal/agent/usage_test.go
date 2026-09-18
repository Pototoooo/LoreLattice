package agent

import (
	"testing"

	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/stretchr/testify/require"
)

func TestRecordUsageAccumulatesEveryModelCall(t *testing.T) {
	engine := &AgentEngine{}
	engine.recordUsage(&types.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, CachedTokens: 80})
	engine.recordUsage(&types.TokenUsage{PromptTokens: 140, CompletionTokens: 30, TotalTokens: 170, CachedTokens: 90})
	engine.recordUsage(nil)
	require.Equal(t, types.TokenUsage{PromptTokens: 240, CompletionTokens: 50, TotalTokens: 290, CachedTokens: 170}, engine.totalUsage)
}
