<template>
  <div class="chunk-detail">
    <div class="info-section">
      <div class="info-field">
        <span class="field-label">片段ID:</span>
        <span class="field-value"><code>{{ data.chunk_id }}</code></span>
      </div>
      <div class="info-field">
        <span class="field-label">文档ID:</span>
        <span class="field-value"><code>{{ data.knowledge_id }}</code></span>
      </div>
      <div class="info-field">
        <span class="field-label">位置:</span>
        <span class="field-value">第 {{ data.chunk_index }} 个片段</span>
      </div>
      <div v-if="data.content_length" class="info-field">
        <span class="field-label">内容长度:</span>
        <span class="field-value">{{ data.content_length }} 字符</span>
      </div>
    </div>

    <div class="info-section">
      <div class="info-section-title">完整内容</div>
      <div class="full-content">{{ data.content }}</div>
    </div>

    <div class="info-section">
      <div class="action-buttons">
        <button class="action-button" @click="copyToClipboard">
          📋 复制内容
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue';
import type { ChunkDetailData } from '@/types/tool-results';

const props = defineProps<{
  data: ChunkDetailData;
}>();

const copyToClipboard = () => {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(props.data.content);
  }
};
</script>

<style lang="less" scoped>
@import './tool-results.less';

.chunk-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0;
}

code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  background: #f0f0f0;
  padding: 2px 4px;
  border-radius: 3px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}
</style>

