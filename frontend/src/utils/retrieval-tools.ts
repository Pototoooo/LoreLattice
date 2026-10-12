export const RETRIEVAL_ALIASES: Record<string, string> = {
  knowledge_search: 'search_knowledge', grep_chunks: 'search_knowledge',
  list_knowledge_chunks: 'read_document', wiki_read_source_doc: 'read_document',
  get_document_info: 'list_documents',
}
export function normalizeRetrievalTools(names: string[]): string[] {
  return [...new Set(names.map(name => RETRIEVAL_ALIASES[name] || name))]
}
