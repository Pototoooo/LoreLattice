/**
 * Tool Results Type Definitions
 * TypeScript interfaces for all tool result types
 */

// Relevance levels
export type RelevanceLevel = '高相关' | '中相关' | '低相关' | '弱相关';

// Display types
export type DisplayType =
    | 'search_results'
    | 'chunk_detail'
    | 'related_chunks'
    | 'knowledge_base_list'
    | 'document_info'
    | 'graph_query_results'
    | 'thinking'
    | 'plan'
    | 'database_query';

// Search result item
export interface SearchResultItem {
    result_index: number;
    chunk_id: string;
    content: string;
    score: number;
    relevance_level: RelevanceLevel;
    knowledge_id: string;
    knowledge_title: string;
    match_type: string;
}

// Chunk item
export interface ChunkItem {
    index: number;
    chunk_id: string;
    chunk_index: number;
    content: string;
    knowledge_id: string;
}

// Knowledge base item
export interface KnowledgeBaseItem {
    index: number;
    id: string;
    name: string;
    description: string;
}

// Graph config
export interface GraphConfig {
    nodes: string[];
    relations: string[];
}

// Search results data
export interface SearchResultsData {
    display_type: 'search_results';
    results?: SearchResultItem[];
    count?: number;
    kb_counts?: Record<string, number>;
    query?: string;
    knowledge_base_id?: string;
}

// Chunk detail data
export interface ChunkDetailData {
    display_type: 'chunk_detail';
    chunk_id: string;
    content: string;
    chunk_index: number;
    knowledge_id: string;
    content_length?: number;
}

// Related chunks data
export interface RelatedChunksData {
    display_type: 'related_chunks';
    chunk_id: string;
    relation_type: string;
    count: number;
    chunks: ChunkItem[];
}

// Knowledge base list data
export interface KnowledgeBaseListData {
    display_type: 'knowledge_base_list';
    knowledge_bases: KnowledgeBaseItem[];
    count: number;
}

// Document info data
export interface DocumentInfoData {
    display_type: 'document_info';
    knowledge_id: string;
    title: string;
    chunk_count_min: number;
}

// Graph query results data
export interface GraphQueryResultsData {
    display_type: 'graph_query_results';
    results: SearchResultItem[];
    count: number;
    graph_config: GraphConfig;
}

// Thinking data
export interface ThinkingData {
    display_type: 'thinking';
    thought: string;
}

// Plan step
export interface PlanStep {
    id: string;
    description: string;
    tools_to_use?: string;
    status: 'pending' | 'in_progress' | 'completed' | 'skipped';
}

// Plan data
export interface PlanData {
    display_type: 'plan';
    task: string;
    steps: PlanStep[];
    total_steps: number;
}

// Database query data
export interface DatabaseQueryData {
    display_type: 'database_query';
    columns: string[];
    rows: Array<Record<string, any>>;
    row_count: number;
    query: string;
}

// Union type for all tool result data
export type ToolResultData =
    | SearchResultsData
    | ChunkDetailData
    | RelatedChunksData
    | KnowledgeBaseListData
    | DocumentInfoData
    | GraphQueryResultsData
    | ThinkingData
    | PlanData
    | DatabaseQueryData;

// Action data (from index.vue)
export interface ActionData {
    description: string;
    success: boolean;
    tool_name?: string;
    arguments?: any;
    output?: string;
    error?: string;
    details?: boolean;
    display_type?: DisplayType;
    tool_data?: Record<string, any>;
}

