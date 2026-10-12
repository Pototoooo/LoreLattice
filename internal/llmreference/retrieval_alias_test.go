package llmreference

import (
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRetrievalGenericReadID(t *testing.T) {
	r := NewRegistry()
	r.RegisterDocument("doc-uuid")
	r.RegisterChunk(ChunkReference{ChunkID: "chunk-uuid"})
	calls := []types.LLMToolCall{{Function: types.FunctionCall{Name: "read_document", Arguments: `{"id":"d1","query":"c1"}`}}, {Function: types.FunctionCall{Name: "read_document", Arguments: `{"id":"c1"}`}}, {Function: types.FunctionCall{Name: "mcp_fixture", Arguments: `{"id":"d1"}`}}}
	r.DecodeToolCalls(calls)
	require.JSONEq(t, `{"id":"doc-uuid","query":"c1"}`, calls[0].Function.Arguments)
	require.JSONEq(t, `{"id":"chunk-uuid"}`, calls[1].Function.Arguments)
	require.JSONEq(t, `{"id":"d1"}`, calls[2].Function.Arguments)
}
