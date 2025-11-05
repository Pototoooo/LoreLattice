<template>
  <div class="document-info">
    <div class="info-card">
      <div class="info-section">
        <div class="info-field">
          <span class="field-label">文档ID:</span>
          <span class="field-value"><code>{{ data.knowledge_id }}</code></span>
        </div>
        <div class="info-field">
          <span class="field-label">文档标题:</span>
          <span class="field-value">{{ data.title }}</span>
        </div>
        <div class="info-field">
          <span class="field-label">片段数量:</span>
          <span class="field-value">至少 {{ data.chunk_count_min }} 个</span>
        </div>
      </div>

      <div class="status-section">
        <div class="status-title">状态说明</div>
        <div class="status-list">
          <div class="status-item">✓ 文档已索引并可搜索</div>
          <div class="status-item">✓ 可使用搜索工具查找文档内容</div>
          <div class="status-item">✓ 可使用 get_chunk_detail 查看片段详情</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue';
import type { DocumentInfoData } from '@/types/tool-results';

const props = defineProps<{
  data: DocumentInfoData;
}>();
</script>

<style lang="less" scoped>
@import './tool-results.less';

.document-info {
  padding: 8px 0;
}

.info-card {
  background: @card-bg;
  border: 1px solid @card-border;
  border-radius: @card-radius;
  padding: 12px;
}

.status-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid @card-border;
}

.status-title {
  font-size: 12px;
  font-weight: 600;
  color: #666;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
}

.status-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.status-item {
  font-size: 12px;
  color: #00a870;
  line-height: 1.5;
}

code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  background: #f0f0f0;
  padding: 2px 4px;
  border-radius: 3px;
}
</style>

