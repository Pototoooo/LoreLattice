package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/Pototoooo/lorelattice/internal/types/interfaces"
)

const (
	ToolSearchKnowledge = "search_knowledge"
	ToolReadDocument    = "read_document"
	ToolListDocuments   = "list_documents"
)

func CanonicalRetrievalTool(name string) string {
	switch name {
	case ToolKnowledgeSearch, ToolGrepChunks:
		return ToolSearchKnowledge
	case ToolListKnowledgeChunks, ToolWikiReadSourceDoc:
		return ToolReadDocument
	case ToolGetDocumentInfo:
		return ToolListDocuments
	default:
		return name
	}
}
func NormalizeRetrievalTools(names []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = CanonicalRetrievalTool(name)
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// Translate legacy calls before canonical schema validation. Legacy regex grep
// becomes literal indexed keyword retrieval; it never executes a database regex.
func legacyRetrievalArgs(name string, args json.RawMessage) json.RawMessage {
	var m map[string]interface{}
	if json.Unmarshal(args, &m) != nil {
		return args
	}
	switch name {
	case ToolKnowledgeSearch:
		if _, ok := m["query"]; !ok {
			if qs, ok := m["queries"].([]interface{}); ok {
				var s []string
				for _, q := range qs {
					if v, ok := q.(string); ok {
						s = append(s, v)
					}
				}
				m["query"] = strings.Join(s, " ")
			}
		}
		if _, ok := m["mode"]; !ok {
			m["mode"] = "hybrid"
		}
	case ToolGrepChunks:
		m["mode"] = "keyword"
	case ToolListKnowledgeChunks, ToolWikiReadSourceDoc:
		if _, ok := m["id"]; !ok {
			for _, k := range []string{"faq_id", "chunk_id", "knowledge_id"} {
				if v, ok := m[k]; ok {
					m["id"] = v
					break
				}
			}
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return args
	}
	return out
}

// SearchKnowledgeTool uses the existing indexed engine and output/citation
// pipeline; neither keyword mode nor legacy aliases reach GrepChunksTool.
type SearchKnowledgeTool struct {
	*KnowledgeSearchTool
	BaseTool
}

func NewSearchKnowledgeTool(engine *KnowledgeSearchTool) *SearchKnowledgeTool {
	return &SearchKnowledgeTool{engine, NewBaseTool(ToolSearchKnowledge, "Search authorized knowledge using hybrid, semantic, or literal indexed keyword retrieval. Use returned dN/cN sources with read_document.", json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1},"mode":{"type":"string","enum":["hybrid","semantic","keyword"],"default":"hybrid"},"knowledge_base_ids":{"type":"array","items":{"type":"string"}},"limit":{"type":"integer","minimum":1,"maximum":100}},"required":["query"]}`))}
}
func (t *SearchKnowledgeTool) Name() string                { return t.BaseTool.Name() }
func (t *SearchKnowledgeTool) Description() string         { return t.BaseTool.Description() }
func (t *SearchKnowledgeTool) Parameters() json.RawMessage { return t.BaseTool.Parameters() }
func (t *SearchKnowledgeTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var in struct {
		Query string   `json:"query"`
		Mode  string   `json:"mode"`
		KBs   []string `json:"knowledge_base_ids"`
		Limit int      `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return retrievalError(err)
	}
	in.Query = strings.TrimSpace(in.Query)
	if in.Query == "" {
		return retrievalError(fmt.Errorf("query must not be empty"))
	}
	if in.Mode == "" {
		in.Mode = "hybrid"
	}
	if in.Mode != "hybrid" && in.Mode != "semantic" && in.Mode != "keyword" {
		return retrievalError(fmt.Errorf("unsupported search mode %q", in.Mode))
	}
	if in.Limit == 0 {
		in.Limit = 10
	}
	if in.Limit < 1 || in.Limit > 100 {
		return retrievalError(fmt.Errorf("limit must be 1..100"))
	}
	requested := map[string]bool{}
	for _, id := range in.KBs {
		requested[id] = true
	}
	selected := types.SearchTargets{}
	for _, st := range t.searchTargets {
		if st == nil {
			continue
		}
		if len(in.KBs) == 0 {
			selected = append(selected, st)
			continue
		}
		for _, id := range in.KBs {
			if st.KnowledgeBaseID == id {
				selected = append(selected, st)
				delete(requested, id)
				break
			}
		}
	}
	if len(requested) > 0 {
		return retrievalError(fmt.Errorf("requested knowledge base is outside authorized scope"))
	}
	if len(selected) == 0 {
		return retrievalError(fmt.Errorf("no authorized knowledge scope"))
	}
	var results []*searchResultWithMeta
	kbIDs := selected.GetAllKnowledgeBaseIDs()
	for _, st := range selected {
		kb, err := t.knowledgeBaseService.GetKnowledgeBaseByIDOnly(ctx, st.KnowledgeBaseID)
		if err != nil {
			return retrievalError(err)
		}
		if kb == nil {
			return retrievalError(fmt.Errorf("knowledge base not found"))
		}
		vector, keyword := kb.IsVectorEnabled(), kb.IsKeywordEnabled() && kb.Type != types.KnowledgeBaseTypeFAQ
		if (in.Mode == "semantic" && !vector) || (in.Mode == "keyword" && !keyword) || (in.Mode == "hybrid" && !vector && !keyword) {
			return retrievalError(fmt.Errorf("mode %s unavailable for knowledge base %s", in.Mode, kb.ID))
		}
		vt, kt := 0.6, 0.5
		if t.config != nil && t.config.Conversation != nil {
			vt = t.config.Conversation.VectorThreshold
			kt = t.config.Conversation.KeywordThreshold
		}
		vt, kt = st.RecallThresholds(vt, kt)
		params := types.SearchParams{RequiredMode: in.Mode, QueryText: in.Query, MatchCount: in.Limit, VectorThreshold: vt, KeywordThreshold: kt, DisableVectorMatch: in.Mode == "keyword", DisableKeywordsMatch: in.Mode == "semantic", KnowledgeIDs: st.KnowledgeIDs, TagIDs: st.TagIDs, ScopeTagIDs: st.ScopeTagIDs}
		hits, err := t.knowledgeBaseService.HybridSearch(ctx, st.KnowledgeBaseID, params)
		if err != nil {
			return retrievalError(err)
		}
		for _, hit := range hits {
			if hit == nil {
				continue
			}
			if hit.KnowledgeBaseID != "" && hit.KnowledgeBaseID != st.KnowledgeBaseID {
				continue
			}
			allowed, err := retrievalScopeAllowsDocument(ctx, selected, hit.KnowledgeID, st.KnowledgeBaseID, t.knowledgeService)
			if err != nil {
				return retrievalError(err)
			}
			if !allowed {
				continue
			}
			results = append(results, &searchResultWithMeta{SearchResult: hit, SourceQuery: in.Query, QueryType: in.Mode, KnowledgeBaseID: st.KnowledgeBaseID, KnowledgeBaseType: kb.Type})
		}
	}
	results = t.deduplicateResults(results)
	if len(results) > 0 && (t.rerankModel != nil || t.chatModel != nil) {
		ranked, err := t.rerankResults(ctx, in.Query, results)
		if err == nil {
			results = ranked
		}
	}
	if len(results) > in.Limit {
		results = t.applyMMR(ctx, results, in.Limit, 0.7)
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > in.Limit {
		results = results[:in.Limit]
	}
	r, e := t.formatOutput(ctx, results, kbIDs, []string{in.Query})
	if r != nil && len(results) == 0 {
		r.Output = "No matches in the authorized scope. Try another query or supported mode."
	}
	return r, e
}
func retrievalError(err error) (*types.ToolResult, error) {
	return &types.ToolResult{Success: false, Error: err.Error()}, err
}

// ReadDocumentTool retains the existing scoped FAQ/chunk renderer. Optional
// query/context reads are bounded to 5000 document chunks, never a global scan.
type ReadDocumentTool struct {
	BaseTool
	reader *ListKnowledgeChunksTool
}

func NewReadDocumentTool(ks interfaces.KnowledgeService, cs interfaces.ChunkService, targets types.SearchTargets) *ReadDocumentTool {
	return &ReadDocumentTool{NewBaseTool(ToolReadDocument, "Read an authorized dN document or cN chunk/FAQ, with exact offset/limit. query locates literal text within this document; context adds neighboring chunks (0..10).", json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","minLength":1},"offset":{"type":"integer","minimum":0},"limit":{"type":"integer","minimum":1,"maximum":100},"query":{"type":"string"},"context":{"type":"integer","minimum":0,"maximum":10}},"required":["id"]}`)), NewListKnowledgeChunksTool(ks, cs, targets)}
}
func (t *ReadDocumentTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var in struct {
		ID          string `json:"id"`
		KnowledgeID string `json:"knowledge_id"`
		ChunkID     string `json:"chunk_id"`
		FAQID       string `json:"faq_id"`
		Offset      int    `json:"offset"`
		Limit       int    `json:"limit"`
		Query       string `json:"query"`
		Context     int    `json:"context"`
		Start       int    `json:"start_chunk_index"`
		End         int    `json:"end_chunk_index"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return retrievalError(err)
	}
	if in.Start < 0 || in.End < 0 || (in.End > 0 && in.End < in.Start) || in.Offset < 0 || in.Context < 0 || in.Context > 10 || in.Limit < 0 || in.Limit > 100 {
		return retrievalError(fmt.Errorf("invalid read bounds"))
	}
	if in.Limit == 0 {
		in.Limit = 20
	}
	docID, chunkID := in.KnowledgeID, in.ChunkID
	if in.FAQID != "" {
		chunkID = in.FAQID
	}
	if docID == "" && chunkID == "" {
		if strings.TrimSpace(in.ID) == "" {
			return retrievalError(fmt.Errorf("id is required"))
		}
		k, err := t.reader.knowledgeService.GetKnowledgeByIDOnly(ctx, in.ID)
		if err == nil && k != nil {
			docID = k.ID
		} else {
			chunkID = in.ID
		}
	}
	if chunkID != "" && in.Context == 0 && in.Query == "" {
		r, e := t.reader.executeByChunkID(ctx, chunkID)
		if e != nil || r == nil || !r.Success {
			return r, e
		}
		c, e := t.reader.chunkService.GetChunkByIDOnly(ctx, chunkID)
		if e != nil {
			return retrievalError(e)
		}
		if c == nil {
			return retrievalError(fmt.Errorf("chunk not found"))
		}
		allowed, e := retrievalScopeAllowsDocument(ctx, t.reader.searchTargets, c.KnowledgeID, c.KnowledgeBaseID, t.reader.knowledgeService)
		if e != nil {
			return retrievalError(e)
		}
		if !allowed {
			return retrievalError(fmt.Errorf("chunk outside authorized scope"))
		}
		return r, nil
	}
	anchor := ""
	if chunkID != "" {
		r, err := t.reader.executeByChunkID(ctx, chunkID)
		if err != nil || r == nil || !r.Success {
			return r, err
		}
		c, err := t.reader.chunkService.GetChunkByIDOnly(ctx, chunkID)
		if err != nil {
			return retrievalError(err)
		}
		if c == nil {
			return retrievalError(fmt.Errorf("chunk not found"))
		}
		docID = c.KnowledgeID
		anchor = chunkID
	}
	k, err := t.reader.knowledgeService.GetKnowledgeByIDOnly(ctx, docID)
	if err != nil || k == nil {
		return retrievalError(fmt.Errorf("document not found"))
	}
	allowed, err := retrievalScopeAllowsDocument(ctx, t.reader.searchTargets, k.ID, k.KnowledgeBaseID, t.reader.knowledgeService)
	if err != nil {
		return retrievalError(err)
	}
	if !allowed {
		return retrievalError(fmt.Errorf("document outside authorized scope"))
	}
	if in.Query == "" && anchor == "" && in.Start == 0 && in.End == 0 {
		b, _ := json.Marshal(ListKnowledgeChunksInput{KnowledgeID: docID, Offset: in.Offset, Limit: in.Limit})
		return t.reader.Execute(ctx, b)
	}
	var chunks []*types.Chunk
	for page := 1; page <= 51; page++ {
		batch, total, err := t.reader.chunkService.GetRepository().ListPagedChunksByKnowledgeID(ctx, k.TenantID, k.ID, &types.Pagination{Page: page, PageSize: 100}, []types.ChunkType{types.ChunkTypeText, types.ChunkTypeFAQ}, nil, "", "", "asc", "")
		if err != nil {
			return retrievalError(err)
		}
		if total > 5000 {
			return retrievalError(fmt.Errorf("document query/context exceeds 5000-chunk bound; use offset/limit reading"))
		}
		chunks = append(chunks, batch...)
		if len(chunks) >= int(total) || len(batch) == 0 {
			break
		}
	}
	selected := map[int]bool{}
	for i, c := range chunks {
		if (anchor != "" && c.ID == anchor) || (in.Query != "" && strings.Contains(strings.ToLower(c.Content), strings.ToLower(in.Query))) || (in.Start > 0 && c.ChunkIndex >= in.Start && (in.End == 0 || c.ChunkIndex <= in.End)) {
			for j := max(0, i-in.Context); j <= min(len(chunks)-1, i+in.Context); j++ {
				selected[j] = true
			}
		}
	}
	hits := []*types.Chunk{}
	for i, c := range chunks {
		if selected[i] {
			hits = append(hits, c)
		}
	}
	total := len(hits)
	if in.Offset >= len(hits) {
		hits = nil
	} else {
		hits = hits[in.Offset:min(len(hits), in.Offset+in.Limit)]
	}
	data := []map[string]interface{}{}
	for _, c := range hits {
		d := map[string]interface{}{"chunk_id": c.ID, "knowledge_id": k.ID, "knowledge_base": k.KnowledgeBaseID, "chunk_index": c.ChunkIndex, "content": c.Content, "chunk_type": c.ChunkType}
		appendFAQChunkData(d, c)
		data = append(data, d)
	}
	return &types.ToolResult{Success: true, Output: t.reader.buildOutput(k.ID, k.Title, int64(total), len(hits), hits), Data: map[string]interface{}{"display_type": "knowledge_chunks_list", "knowledge_id": k.ID, "knowledge_title": k.Title, "chunks": data, "total_chunks": total, "fetched_chunks": len(hits), "offset": in.Offset, "limit": in.Limit}}, nil
}

type ListDocumentsTool struct {
	BaseTool
	ks         interfaces.KnowledgeService
	targets    types.SearchTargets
	legacyInfo types.Tool
}

func NewListDocumentsTool(ks interfaces.KnowledgeService, cs interfaces.ChunkService, targets types.SearchTargets) *ListDocumentsTool {
	return &ListDocumentsTool{NewBaseTool(ToolListDocuments, "List documents within one authorized knowledge base, preserving current file/tag scope. Use returned document IDs with read_document.", json.RawMessage(`{"type":"object","properties":{"knowledge_base_id":{"type":"string"},"query":{"type":"string"},"offset":{"type":"integer","minimum":0},"limit":{"type":"integer","minimum":1,"maximum":100},"knowledge_ids":{"type":"array","items":{"type":"string"}},"faq_ids":{"type":"array","items":{"type":"string"}}}`)), ks, targets, &scopedLegacyDocumentInfo{NewGetDocumentInfoTool(ks, cs, targets)}}
}
func (t *ListDocumentsTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var in struct {
		KB     string   `json:"knowledge_base_id"`
		Query  string   `json:"query"`
		Offset int      `json:"offset"`
		Limit  int      `json:"limit"`
		IDs    []string `json:"knowledge_ids"`
		FAQs   []string `json:"faq_ids"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return retrievalError(err)
	}
	if in.Offset < 0 || in.Limit < 0 || in.Limit > 100 {
		return retrievalError(fmt.Errorf("invalid list bounds"))
	}
	if in.Limit == 0 {
		in.Limit = 20
	}
	if in.KB == "" {
		ids := t.targets.GetAllKnowledgeBaseIDs()
		if len(ids) != 1 {
			return retrievalError(fmt.Errorf("select one authorized knowledge_base_id"))
		}
		in.KB = ids[0]
	}
	if !t.targets.ContainsKB(in.KB) {
		return retrievalError(fmt.Errorf("knowledge base outside authorized scope"))
	}
	scoped := types.SearchTargets{}
	for _, st := range t.targets {
		if st != nil && st.KnowledgeBaseID == in.KB {
			scoped = append(scoped, st)
		}
	}
	// Page in the storage layer, then enforce @file/@tag scope before exposing metadata.
	documents := []map[string]interface{}{}
	cursor := 0
	more := true
	offset := in.Offset
	for more && len(documents) < in.Limit {
		batch, hasMore, _, err := t.ks.SearchKnowledgeForScopes(ctx, []types.KnowledgeSearchScope{{TenantID: scoped.GetTenantIDForKB(in.KB), KBID: in.KB}}, in.Query, cursor, 100, nil)
		if err != nil {
			return retrievalError(err)
		}
		more = hasMore
		if len(batch) == 0 {
			break
		}
		cursor += len(batch)
		for _, k := range batch {
			if k == nil || k.KnowledgeBaseID != in.KB {
				continue
			}
			ok, err := retrievalScopeAllowsDocument(ctx, scoped, k.ID, k.KnowledgeBaseID, t.ks)
			if err != nil {
				return retrievalError(err)
			}
			if !ok {
				continue
			}
			if len(in.IDs) > 0 {
				match := false
				for _, id := range in.IDs {
					if id == k.ID {
						match = true
					}
				}
				if !match {
					continue
				}
			}
			if offset > 0 {
				offset--
				continue
			}
			if len(documents) == in.Limit {
				break
			}
			documents = append(documents, map[string]interface{}{"knowledge_id": k.ID, "knowledge_base_id": k.KnowledgeBaseID, "title": k.Title, "parse_status": k.ParseStatus})
		}
	}
	b, _ := json.Marshal(documents)
	return &types.ToolResult{Success: true, Output: string(b), Data: map[string]interface{}{"display_type": "document_info", "documents": documents, "count": len(documents), "offset": in.Offset, "limit": in.Limit, "next_offset": in.Offset + len(documents)}}, nil
}

// Each target is an OR branch; its document pins and tag/share constraints are
// intersected. A requested doc must not escape ScopeTagIDs via a full-KB target.
func retrievalScopeAllowsDocument(ctx context.Context, targets types.SearchTargets, id, kb string, ks interfaces.KnowledgeService) (bool, error) {
	for _, st := range targets {
		if st == nil || st.KnowledgeBaseID != kb {
			continue
		}
		if st.Type == types.SearchTargetTypeKnowledge {
			found := false
			for _, pin := range st.KnowledgeIDs {
				if pin == id {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if len(st.TagIDs) == 0 && len(st.ScopeTagIDs) == 0 {
			return true, nil
		}
		tags, err := ks.GetKnowledgeTags(ctx, []string{id})
		if err != nil {
			return false, err
		}
		matches := func(wanted []string) bool {
			if len(wanted) == 0 {
				return true
			}
			for _, tag := range tags[id] {
				if tag == nil {
					continue
				}
				for _, want := range wanted {
					if tag.ID == want {
						return true
					}
				}
			}
			return false
		}
		if matches(st.TagIDs) && matches(st.ScopeTagIDs) {
			return true, nil
		}
	}
	return false, nil
}

// The hidden batch-metadata alias keeps its old schema and partial-result
// behavior, but applies the same scope-level tag constraints as canonical tools.
type scopedLegacyDocumentInfo struct{ *GetDocumentInfoTool }

func (t *scopedLegacyDocumentInfo) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var in GetDocumentInfoInput
	if err := json.Unmarshal(args, &in); err != nil {
		return retrievalError(err)
	}
	check := func(id, kb string) error {
		ok, err := retrievalScopeAllowsDocument(ctx, t.searchTargets, id, kb, t.knowledgeService)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("document outside authorized scope")
		}
		return nil
	}
	for _, id := range in.KnowledgeIDs {
		k, err := t.knowledgeService.GetKnowledgeByIDOnly(ctx, id)
		if err != nil || k == nil {
			continue
		}
		if err := check(k.ID, k.KnowledgeBaseID); err != nil {
			return retrievalError(err)
		}
	}
	for _, id := range in.FAQIDs {
		c, err := t.chunkService.GetChunkByIDOnly(ctx, id)
		if err != nil || c == nil {
			continue
		}
		if err := check(c.KnowledgeID, c.KnowledgeBaseID); err != nil {
			return retrievalError(err)
		}
	}
	return t.GetDocumentInfoTool.Execute(ctx, args)
}
