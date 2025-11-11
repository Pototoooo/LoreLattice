<template>
  <div class="tool-result-renderer">
    <!-- Search Results -->
    <SearchResults 
      v-if="displayType === 'search_results'" 
      :data="toolData as SearchResultsData" 
      :arguments="toolArguments"
    />
    
    <!-- Chunk Detail -->
    <ChunkDetail 
      v-else-if="displayType === 'chunk_detail'" 
      :data="toolData as ChunkDetailData" 
    />
    
    <!-- Related Chunks -->
    <RelatedChunks 
      v-else-if="displayType === 'related_chunks'" 
      :data="toolData as RelatedChunksData" 
    />
    
    <!-- Knowledge Base List -->
    <KnowledgeBaseList 
      v-else-if="displayType === 'knowledge_base_list'" 
      :data="toolData as KnowledgeBaseListData" 
    />
    
    <!-- Document Info -->
    <DocumentInfo 
      v-else-if="displayType === 'document_info'" 
      :data="toolData as DocumentInfoData" 
    />
    
    <!-- Graph Query Results -->
    <GraphQueryResults 
      v-else-if="displayType === 'graph_query_results'" 
      :data="toolData as GraphQueryResultsData" 
    />
    
    <!-- Thinking Display -->
    <ThinkingDisplay 
      v-else-if="displayType === 'thinking'" 
      :data="toolData as ThinkingData" 
    />
    
    <!-- Plan Display -->
    <PlanDisplay 
      v-else-if="displayType === 'plan'" 
      :data="toolData as PlanData" 
    />
    
    <!-- Database Query Display -->
    <DatabaseQuery 
      v-else-if="displayType === 'database_query'" 
      :data="toolData as DatabaseQueryData" 
    />
    
    <!-- Web Search Results Display -->
    <WebSearchResults 
      v-else-if="displayType === 'web_search_results'" 
      :data="toolData as WebSearchResultsData" 
    />
    
    <!-- Fallback: Display raw output -->
    <div v-else class="fallback-output">
      <div class="detail-output">{{ output }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps, computed } from 'vue';
import type { 
  DisplayType,
  SearchResultsData,
  ChunkDetailData,
  RelatedChunksData,
  KnowledgeBaseListData,
  DocumentInfoData,
  GraphQueryResultsData,
  ThinkingData,
  PlanData,
  DatabaseQueryData,
  WebSearchResultsData
} from '@/types/tool-results';

import SearchResults from './tool-results/SearchResults.vue';
import ChunkDetail from './tool-results/ChunkDetail.vue';
import RelatedChunks from './tool-results/RelatedChunks.vue';
import KnowledgeBaseList from './tool-results/KnowledgeBaseList.vue';
import DocumentInfo from './tool-results/DocumentInfo.vue';
import GraphQueryResults from './tool-results/GraphQueryResults.vue';
import ThinkingDisplay from './tool-results/ThinkingDisplay.vue';
import PlanDisplay from './tool-results/PlanDisplay.vue';
import DatabaseQuery from './tool-results/DatabaseQuery.vue';
import WebSearchResults from './tool-results/WebSearchResults.vue';

interface Props {
  displayType?: DisplayType;
  toolData?: Record<string, any>;
  output?: string;
  arguments?: Record<string, any>;
}

const props = defineProps<Props>();

const displayType = computed(() => props.displayType);
const toolData = computed(() => props.toolData || {});
const output = computed(() => props.output || '');
const toolArguments = computed(() => props.arguments || {});
</script>

<style lang="less" scoped>
.tool-result-renderer {
  margin: 8px 0;
}

.fallback-output {
  .detail-output {
    font-size: 13px;
    color: #333;
    background: #f5f5f5;
    padding: 8px;
    border-radius: 4px;
    white-space: pre-wrap;
    word-break: break-word;
    border: 1px solid #e7e7e7;
  }
}
</style>

