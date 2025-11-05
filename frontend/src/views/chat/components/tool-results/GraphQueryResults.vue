<template>
  <div class="graph-query-results">
    <!-- Graph Configuration Card -->
    <div v-if="data.graph_config" class="stats-card">
      <div class="stats-title">图谱配置</div>
      <div class="info-field">
        <span class="field-label">实体类型:</span>
        <span class="field-value">{{ data.graph_config.nodes.join(', ') }}</span>
      </div>
      <div class="info-field">
        <span class="field-label">关系类型:</span>
        <span class="field-value">{{ data.graph_config.relations.join(', ') }}</span>
      </div>
    </div>

    <!-- Results List -->
    <div v-if="data.results && data.results.length > 0" class="results-list">
      <div class="results-header">
        找到 {{ data.count }} 条相关结果
      </div>
      
      <div 
        v-for="result in data.results" 
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
            <span class="score">{{ (result.score * 100).toFixed(0) }}%</span>
            <span class="expand-icon" :class="{ expanded: expandedResults.includes(result.chunk_id) }">
              ▶
            </span>
          </div>
        </div>
        
        <div class="result-content" :class="{ expanded: expandedResults.includes(result.chunk_id) }">
          <div class="full-content">{{ result.content }}</div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      未找到相关的图谱信息
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, defineProps } from 'vue';
import type { GraphQueryResultsData, RelevanceLevel } from '@/types/tool-results';

const props = defineProps<{
  data: GraphQueryResultsData;
}>();

const expandedResults = ref<string[]>([]);

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

.graph-query-results {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.results-header {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  padding: 4px 0;
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
</style>

