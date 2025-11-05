<template>
  <div class="search-results">
    <!-- Knowledge Base Statistics Card -->
    <div v-if="kbCounts && Object.keys(kbCounts).length > 0" class="stats-card">
      <div class="stats-title">知识库覆盖</div>
      <div class="stats-list">
        <div v-for="(count, kbId) in kbCounts" :key="kbId" class="stats-item">
          <span>{{ kbId }}</span>
          <span>{{ count }} 条结果</span>
        </div>
      </div>
    </div>

    <!-- Search Results List -->
    <div v-if="results && results.length > 0" class="results-list">
      <div 
        v-for="result in results" 
        :key="result.chunk_id"
        class="result-card"
      >
        <div class="result-header" @click="toggleResult(result.chunk_id)">
          <div class="result-title">
            <span class="result-index">#{{ result.result_index }}</span>
            <span class="relevance-badge" :class="getRelevanceClass(result.relevance_level)">
              {{ result.relevance_level }}
            </span>
            <span class="knowledge-title">{{ result.knowledge_title }}</span>
          </div>
          <div class="result-meta">
            <span class="match-type-badge">
              {{ getMatchTypeIcon(result.match_type) }} {{ result.match_type }}
            </span>
            <span class="score">{{ (result.score * 100).toFixed(0) }}%</span>
            <span class="expand-icon" :class="{ expanded: expandedResults.includes(result.chunk_id) }">
              ▶
            </span>
          </div>
        </div>
        
        <div class="result-content" :class="{ expanded: expandedResults.includes(result.chunk_id) }">
          <div class="info-section">
            <div class="info-section-title">内容</div>
            <div class="full-content">{{ result.content }}</div>
          </div>
          
          <div class="info-section">
            <div class="info-field">
              <span class="field-label">片段ID:</span>
              <span class="field-value"><code>{{ result.chunk_id }}</code></span>
            </div>
            <div class="info-field">
              <span class="field-label">文档ID:</span>
              <span class="field-value"><code>{{ result.knowledge_id }}</code></span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state">
      没有找到搜索结果
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, defineProps, computed } from 'vue';
import type { SearchResultsData, SearchResultItem, RelevanceLevel } from '@/types/tool-results';
import { getMatchTypeIcon } from '@/utils/tool-icons';

const props = defineProps<{
  data: SearchResultsData;
}>();

const expandedResults = ref<string[]>([]);

const results = computed(() => props.data.results || []);
const kbCounts = computed(() => props.data.kb_counts);

const toggleResult = (chunkId: string) => {
  const index = expandedResults.value.indexOf(chunkId);
  if (index > -1) {
    expandedResults.value.splice(index, 1);
  } else {
    expandedResults.value.push(chunkId);
  }
};

const getRelevanceClass = (level: RelevanceLevel): string => {
  const classMap: Record<RelevanceLevel, string> = {
    '高相关': 'high',
    '中相关': 'medium',
    '低相关': 'low',
    '弱相关': 'weak',
  };
  return classMap[level] || 'weak';
};
</script>

<style lang="less" scoped>
@import './tool-results.less';

.search-results {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result-index {
  font-size: 13px;
  color: #8b8b8b;
  font-weight: 600;
}

.knowledge-title {
  font-size: 13px;
  color: #333;
  flex: 1;
}

.score {
  font-size: 12px;
  color: #8b8b8b;
  font-weight: 500;
}

code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  background: #f0f0f0;
  padding: 2px 4px;
  border-radius: 3px;
}
</style>

