package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentCompleteDataIncludesCumulativeTokenUsage(t *testing.T) {
	payload, err := json.Marshal(AgentCompleteData{PromptTokens: 240, CompletionTokens: 50, TotalTokens: 290, CachedTokens: 170})
	require.NoError(t, err)
	require.JSONEq(t, `{"prompt_tokens":240,"completion_tokens":50,"total_tokens":290,"cached_tokens":170,"session_id":"","total_steps":0,"final_answer":"","total_duration_ms":0}`, string(payload))
}
