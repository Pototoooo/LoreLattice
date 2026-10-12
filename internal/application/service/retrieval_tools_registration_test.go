package service

import (
	"context"
	"github.com/Pototoooo/lorelattice/internal/agent/tools"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/Pototoooo/lorelattice/internal/types/interfaces"
	"github.com/stretchr/testify/require"
	"testing"
)

type retrievalRegistrationKB struct {
	interfaces.KnowledgeBaseService
	wiki bool
}

func (s *retrievalRegistrationKB) GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: "kb", IndexingStrategy: types.IndexingStrategy{VectorEnabled: !s.wiki, KeywordEnabled: !s.wiki, WikiEnabled: s.wiki}}, nil
}
func TestRetrievalRegistrationCanonicalOnly(t *testing.T) {
	s := &agentService{knowledgeBaseService: &retrievalRegistrationKB{}}
	r := tools.NewToolRegistry()
	c := &types.AgentConfig{AllowedTools: []string{tools.ToolKnowledgeSearch, tools.ToolGrepChunks, tools.ToolListKnowledgeChunks, tools.ToolGetDocumentInfo}, SearchTargets: types.SearchTargets{{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb", TenantID: 42}}}
	require.NoError(t, s.registerTools(context.Background(), r, c, nil, nil, "session"))
	require.Equal(t, []string{tools.ToolListDocuments, tools.ToolReadDocument, tools.ToolSearchKnowledge}, r.ListTools())
}
func TestRetrievalWikiScopeRetainsCanonicalReaders(t *testing.T) {
	s := &agentService{knowledgeBaseService: &retrievalRegistrationKB{wiki: true}}
	r := tools.NewToolRegistry()
	c := &types.AgentConfig{AllowedTools: []string{tools.ToolReadDocument, tools.ToolListDocuments, tools.ToolSearchKnowledge}, SearchTargets: types.SearchTargets{{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb", TenantID: 42}}}
	require.NoError(t, s.registerTools(context.Background(), r, c, nil, nil, "session"))
	require.Equal(t, []string{tools.ToolListDocuments, tools.ToolReadDocument}, r.ListTools())
}
