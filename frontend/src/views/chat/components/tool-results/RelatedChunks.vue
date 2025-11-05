<template>
  <div class="related-chunks">
    <div class="info-section">
      <div class="info-field">
        <span class="field-label">参考片段:</span>
        <span class="field-value"><code>{{ data.chunk_id }}</code></span>
      </div>
      <div class="info-field">
        <span class="field-label">关系类型:</span>
        <span class="field-value">{{ relationTypeLabel }}</span>
      </div>
      <div class="info-field">
        <span class="field-label">相关片段数:</span>
        <span class="field-value">{{ data.count }} 个</span>
      </div>
    </div>

    <div v-if="data.chunks && data.chunks.length > 0" class="chunks-list">
      <div 
        v-for="chunk in data.chunks" 
        :key="chunk.chunk_id"
        class="result-card"
      >
        <div class="result-header" @click="toggleChunk(chunk.chunk_id)">
          <div class="result-title">
            <span class="chunk-index">片段 #{{ chunk.index }}</span>
            <span class="chunk-position">(位置: {{ chunk.chunk_index }})</span>
          </div>
          <span class="expand-icon" :class="{ expanded: expandedChunks.includes(chunk.chunk_id) }">
            ▶
          </span>
        </div>
        
        <div class="result-content" :class="{ expanded: expandedChunks.includes(chunk.chunk_id) }">
          <div class="info-field">
            <span class="field-label">片段ID:</span>
            <span class="field-value"><code>{{ chunk.chunk_id }}</code></span>
          </div>
          <div class="info-section">
            <div class="info-section-title">内容</div>
            <div class="full-content">{{ chunk.content }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      没有找到相关片段
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, defineProps, computed } from 'vue';
import type { RelatedChunksData } from '@/types/tool-results';

const props = defineProps<{
  data: RelatedChunksData;
}>();

const expandedChunks = ref<string[]>([]);

const relationTypeLabel = computed(() => {
  const labels: Record<string, string> = {
    'sequential': '顺序相关',
    'semantic': '语义相关',
  };
  return labels[props.data.relation_type] || props.data.relation_type;
});

const toggleChunk = (chunkId: string) => {
  const index = expandedChunks.value.indexOf(chunkId);
  if (index > -1) {
    expandedChunks.value.splice(index, 1);
  } else {
    expandedChunks.value.push(chunkId);
  }
};
</script>

<style lang="less" scoped>
@import './tool-results.less';

.related-chunks {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chunks-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.chunk-index {
  font-size: 13px;
  color: #333;
  font-weight: 600;
}

.chunk-position {
  font-size: 12px;
  color: #8b8b8b;
}

code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  background: #f0f0f0;
  padding: 2px 4px;
  border-radius: 3px;
}
</style>

