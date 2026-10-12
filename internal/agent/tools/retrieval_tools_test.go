package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/Pototoooo/lorelattice/internal/application/repository"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/Pototoooo/lorelattice/internal/types/interfaces"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type retrievalFixtureKB struct {
	interfaces.KnowledgeBaseService
	kb     *types.KnowledgeBase
	params []types.SearchParams
	hits   []*types.SearchResult
	err    error
}

func (s *retrievalFixtureKB) GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error) {
	return s.kb, nil
}
func (s *retrievalFixtureKB) HybridSearch(_ context.Context, _ string, p types.SearchParams) ([]*types.SearchResult, error) {
	s.params = append(s.params, p)
	if s.err != nil {
		return nil, s.err
	}
	hits := append([]*types.SearchResult(nil), s.hits...)
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > p.MatchCount {
		hits = hits[:p.MatchCount]
	}
	return hits, nil
}

type retrievalFixtureKS struct {
	interfaces.KnowledgeService
	docs map[string]*types.Knowledge
	tags map[string][]*types.KnowledgeTag
}

func (s *retrievalFixtureKS) GetKnowledgeByIDOnly(_ context.Context, id string) (*types.Knowledge, error) {
	if d := s.docs[id]; d != nil {
		return d, nil
	}
	return nil, errors.New("not found")
}
func (s *retrievalFixtureKS) GetKnowledgeTags(context.Context, []string) (map[string][]*types.KnowledgeTag, error) {
	return s.tags, nil
}
func (s *retrievalFixtureKS) SearchKnowledgeForScopes(_ context.Context, scopes []types.KnowledgeSearchScope, _ string, offset, limit int, _ []string) ([]*types.Knowledge, bool, int64, error) {
	var docs []*types.Knowledge
	for _, d := range s.docs {
		if d.KnowledgeBaseID == scopes[0].KBID && d.TenantID == scopes[0].TenantID {
			docs = append(docs, d)
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
	total := len(docs)
	if offset >= total {
		return nil, false, int64(total), nil
	}
	end := min(total, offset+limit)
	return docs[offset:end], end < total, int64(total), nil
}

type retrievalFixtureCS struct {
	interfaces.ChunkService
	repo interfaces.ChunkRepository
}

func (s *retrievalFixtureCS) GetRepository() interfaces.ChunkRepository { return s.repo }
func (s *retrievalFixtureCS) GetChunkByIDOnly(ctx context.Context, id string) (*types.Chunk, error) {
	return s.repo.GetChunkByIDOnly(ctx, id)
}
func retrievalFixture(t *testing.T) (*retrievalFixtureKB, *retrievalFixtureKS, *retrievalFixtureCS, types.SearchTargets) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	require.NoError(t, db.AutoMigrate(&types.Chunk{}))
	for i := 0; i < 12; i++ {
		content := fmt.Sprintf("paragraph %d", i)
		if i == 7 {
			content = "needle_literal"
		}
		c := &types.Chunk{ID: fmt.Sprintf("chunk-%02d", i), TenantID: 42, KnowledgeID: "doc-1", KnowledgeBaseID: "kb-1", ChunkIndex: i, Content: content, ChunkType: types.ChunkTypeText, Status: int(types.ChunkStatusIndexed)}
		require.NoError(t, db.Create(c).Error)
	}
	kb := &retrievalFixtureKB{kb: &types.KnowledgeBase{ID: "kb-1", TenantID: 42, Type: types.KnowledgeBaseTypeDocument, IndexingStrategy: types.IndexingStrategy{VectorEnabled: true, KeywordEnabled: true}}}
	ks := &retrievalFixtureKS{docs: map[string]*types.Knowledge{"doc-1": {ID: "doc-1", Title: "fixture", KnowledgeBaseID: "kb-1", TenantID: 42}}}
	cs := &retrievalFixtureCS{repo: repository.NewChunkRepository(db)}
	targets := types.SearchTargets{{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 42}}
	return kb, ks, cs, targets
}
func TestRetrievalCanonicalSurface(t *testing.T) {
	names := NormalizeRetrievalTools([]string{ToolKnowledgeSearch, ToolGrepChunks, ToolListKnowledgeChunks, ToolGetDocumentInfo, ToolWikiReadSourceDoc})
	require.Equal(t, []string{ToolSearchKnowledge, ToolReadDocument, ToolListDocuments}, names)
	defs := AvailableToolDefinitions()
	for _, d := range defs {
		require.Equal(t, d.Name, CanonicalRetrievalTool(d.Name))
	}
}
func TestRetrievalIndexedModesAndOldestOf600(t *testing.T) {
	kb, ks, cs, targets := retrievalFixture(t)
	for i := 0; i < 600; i++ {
		score := 0.01
		content := fmt.Sprintf("candidate %d", i)
		if i == 0 {
			score = 1
			content = "oldest relevant source"
		}
		kb.hits = append(kb.hits, &types.SearchResult{ID: fmt.Sprint(i), KnowledgeID: "doc-1", KnowledgeBaseID: "kb-1", Content: content, ChunkIndex: i, Score: score})
	}
	for _, mode := range []string{"keyword", "semantic", "hybrid"} {
		tool := NewSearchKnowledgeTool(NewKnowledgeSearchTool(kb, ks, cs, targets, nil, nil, nil))
		args := json.RawMessage(fmt.Sprintf(`{"query":"source","mode":%q,"limit":1}`, mode))
		r, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		require.True(t, r.Success)
		require.Contains(t, r.Output, "oldest relevant source")
		p := kb.params[len(kb.params)-1]
		require.Equal(t, mode == "keyword", p.DisableVectorMatch)
		require.Equal(t, mode == "semantic", p.DisableKeywordsMatch)
	}
	// Engine errors are not disguised as successful empty search.
	kb.err = errors.New("index unavailable")
	r, err := NewSearchKnowledgeTool(NewKnowledgeSearchTool(kb, ks, cs, targets, nil, nil, nil)).Execute(context.Background(), json.RawMessage(`{"query":"x"}`))
	require.Error(t, err)
	require.False(t, r.Success)
}
func TestRetrievalModeAndScopeRefusal(t *testing.T) {
	kb, ks, cs, targets := retrievalFixture(t)
	tool := NewSearchKnowledgeTool(NewKnowledgeSearchTool(kb, ks, cs, targets, nil, nil, nil))
	for _, args := range []string{`{"query":"x","mode":"regex"}`, `{"query":"x","limit":101}`, `{"query":"x","knowledge_base_ids":["foreign"]}`, `{"query":" "}`} {
		r, err := tool.Execute(context.Background(), json.RawMessage(args))
		require.Error(t, err)
		require.False(t, r.Success)
	}
	kb.kb.IndexingStrategy.KeywordEnabled = false
	r, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"x","mode":"keyword"}`))
	require.Error(t, err)
	require.False(t, r.Success)
	require.Empty(t, kb.params)
}
func TestRetrievalLegacyCallsHiddenAndTranslated(t *testing.T) {
	kb, ks, cs, targets := retrievalFixture(t)
	registry := NewToolRegistry()
	registry.RegisterTool(NewSearchKnowledgeTool(NewKnowledgeSearchTool(kb, ks, cs, targets, nil, nil, nil)))
	registry.RegisterTool(NewReadDocumentTool(ks, cs, targets))
	registry.RegisterTool(NewListDocumentsTool(ks, cs, targets))
	require.Equal(t, []string{ToolListDocuments, ToolReadDocument, ToolSearchKnowledge}, registry.ListTools())
	for _, call := range []struct{ name, args string }{{ToolKnowledgeSearch, `{"queries":["one","two"]}`}, {ToolGrepChunks, `{"query":"literal"}`}, {ToolListKnowledgeChunks, `{"knowledge_id":"doc-1","offset":3,"limit":4}`}, {ToolGetDocumentInfo, `{"knowledge_ids":["doc-1"]}`}} {
		r, err := registry.ExecuteTool(context.Background(), call.name, json.RawMessage(call.args))
		require.NoError(t, err, call.name)
		require.True(t, r.Success, call.name)
	}
	require.True(t, kb.params[1].DisableVectorMatch)
}
func TestRetrievalReadExactOffsetQueryAndContext(t *testing.T) {
	_, ks, cs, targets := retrievalFixture(t)
	tool := NewReadDocumentTool(ks, cs, targets)
	for _, offset := range []int{0, 3, 7, 11} {
		r, err := tool.Execute(context.Background(), json.RawMessage(fmt.Sprintf(`{"id":"doc-1","offset":%d,"limit":4}`, offset)))
		require.NoError(t, err)
		require.True(t, r.Success)
		chunks := r.Data["chunks"].([]map[string]interface{})
		require.Len(t, chunks, min(4, 12-offset))
		require.Equal(t, offset, chunks[0]["chunk_index"])
	}
	r, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"doc-1","query":"needle_literal","context":1}`))
	require.NoError(t, err)
	require.Len(t, r.Data["chunks"], 3)
	r, err = tool.Execute(context.Background(), json.RawMessage(`{"id":"chunk-07","context":1}`))
	require.NoError(t, err)
	require.Len(t, r.Data["chunks"], 3)
	r, err = tool.Execute(context.Background(), json.RawMessage(`{"id":"doc-1","query":".*"}`))
	require.NoError(t, err)
	require.Empty(t, r.Data["chunks"])
}
func TestRetrievalDocumentPinScope(t *testing.T) {
	_, ks, cs, targets := retrievalFixture(t)
	targets[0].Type = types.SearchTargetTypeKnowledge
	targets[0].KnowledgeIDs = []string{"another-doc"}
	r, err := NewReadDocumentTool(ks, cs, targets).Execute(context.Background(), json.RawMessage(`{"id":"doc-1"}`))
	require.Error(t, err)
	require.False(t, r.Success)
	r, err = NewListDocumentsTool(ks, cs, targets).Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Empty(t, r.Data["documents"])
}

func TestRetrievalShareTagConstraint(t *testing.T) {
	_, ks, _, targets := retrievalFixture(t)
	targets[0].ScopeTagIDs = []string{"allowed"}
	ok, err := retrievalScopeAllowsDocument(context.Background(), targets, "doc-1", "kb-1", ks)
	require.NoError(t, err)
	require.False(t, ok)
	ks.tags = map[string][]*types.KnowledgeTag{"doc-1": {{ID: "allowed"}}}
	ok, err = retrievalScopeAllowsDocument(context.Background(), targets, "doc-1", "kb-1", ks)
	require.NoError(t, err)
	require.True(t, ok)
	targets[0].TagIDs = []string{"selected"}
	ok, err = retrievalScopeAllowsDocument(context.Background(), targets, "doc-1", "kb-1", ks)
	require.NoError(t, err)
	require.False(t, ok)
	ks.tags["doc-1"] = append(ks.tags["doc-1"], &types.KnowledgeTag{ID: "selected"})
	ok, err = retrievalScopeAllowsDocument(context.Background(), targets, "doc-1", "kb-1", ks)
	require.NoError(t, err)
	require.True(t, ok)
}
func TestRetrievalListPaging(t *testing.T) {
	_, ks, cs, targets := retrievalFixture(t)
	for i := 2; i < 130; i++ {
		id := fmt.Sprintf("doc-%03d", i)
		ks.docs[id] = &types.Knowledge{ID: id, TenantID: 42, KnowledgeBaseID: "kb-1", Title: id}
	}
	tool := NewListDocumentsTool(ks, cs, targets)
	seen := map[string]bool{}
	for offset := 0; offset < 129; offset += 13 {
		r, err := tool.Execute(context.Background(), json.RawMessage(fmt.Sprintf(`{"offset":%d,"limit":13}`, offset)))
		require.NoError(t, err)
		for _, d := range r.Data["documents"].([]map[string]interface{}) {
			id := d["knowledge_id"].(string)
			require.False(t, seen[id])
			seen[id] = true
		}
	}
	require.Len(t, seen, 129)
}
func TestRetrievalFAQRead(t *testing.T) {
	_, ks, cs, targets := retrievalFixture(t)
	raw, _ := json.Marshal(types.FAQChunkMetadata{StandardQuestion: "FAQ question", Answers: []string{"FAQ answer"}})
	c := &types.Chunk{ID: "faq-1", TenantID: 42, KnowledgeID: "doc-1", KnowledgeBaseID: "kb-1", ChunkType: types.ChunkTypeFAQ, Content: "FAQ question", Metadata: types.JSON(raw), Status: int(types.ChunkStatusIndexed)}
	require.NoError(t, cs.repo.CreateChunks(context.Background(), []*types.Chunk{c}))
	r, err := NewReadDocumentTool(ks, cs, targets).Execute(context.Background(), json.RawMessage(`{"id":"faq-1"}`))
	require.NoError(t, err)
	require.True(t, r.Success)
	require.Contains(t, r.Output, "FAQ answer")
}
