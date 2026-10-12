# Unified retrieval tools

The model sees search_knowledge, read_document and list_documents instead of five overlapping RAG tools.
Search supports hybrid (default), semantic and keyword modes. Keyword retrieval uses the existing index engine, never the old time-ordered database regex window. Unsupported KB modes and engine failures are errors, not successful empty results.

read_document accepts an authorized document/chunk/FAQ id; offset/limit are exact. query is literal, case-insensitive document-local text; context adds 0..10 neighboring chunks. Query/context reads refuse documents above 5000 text/FAQ chunks; ordinary pagination remains available.

Old stored tool names normalize at registration, without duplicate model definitions. Legacy knowledge_search queries are combined into one hybrid query; grep_chunks query is now literal indexed keyword retrieval, not regex. list_knowledge_chunks keeps its IDs/pagination; get_document_info retains its scoped batched metadata handler. Original historical tool names remain renderable. No stored agent JSON is migrated destructively.

The 600-candidate unit fixture verifies routing to the indexed retriever and relevance ordering without a creation-time window. It is not a live PostgreSQL performance benchmark or a model-quality improvement claim.
