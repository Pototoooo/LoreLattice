import assert from 'node:assert/strict'
import test from 'node:test'
import { normalizeRetrievalTools } from './retrieval-tools.ts'
import { agentRequiresRerankModel } from './agent-readiness.ts'
test('legacy retrieval names collapse to three canonical tools', () => {
 assert.deepEqual(normalizeRetrievalTools(['knowledge_search','grep_chunks','list_knowledge_chunks','wiki_read_source_doc','get_document_info']), ['search_knowledge','read_document','list_documents'])
})
test('unrelated tools stay selected and normalization is idempotent', () => {
 const tools = ['web_search','search_knowledge','read_document','list_documents']
 assert.deepEqual(normalizeRetrievalTools(normalizeRetrievalTools(tools)), tools)
})
test('canonical search has the same rerank readiness requirement', () => {
 assert.equal(agentRequiresRerankModel({kb_selection_mode:'all',allowed_tools:['search_knowledge']}), true)
 assert.equal(agentRequiresRerankModel({kb_selection_mode:'all',allowed_tools:['read_document']}), false)
 assert.equal(agentRequiresRerankModel({kb_selection_mode:'none',allowed_tools:['search_knowledge']}), false)
})
