<template>
  <div class="plan-display">
    <div v-if="data.task" class="plan-task">
      <strong>任务:</strong> {{ data.task }}
    </div>
    
    <div v-if="data.steps && data.steps.length > 0" class="plan-steps">
      <div class="steps-header">计划步骤 (共 {{ data.total_steps || data.steps.length }} 步):</div>
      <div v-for="(step, index) in data.steps" :key="step.id || index" class="step-item">
        <div class="step-header">
          <span class="step-number">{{ index + 1 }}.</span>
          <span class="step-status" :class="`status-${step.status}`">
            {{ getStatusEmoji(step.status) }}
          </span>
          <span class="step-status-text" :class="`status-${step.status}`">
            [{{ getStatusText(step.status) }}]
          </span>
          <span class="step-description">{{ step.description }}</span>
        </div>
        <div v-if="step.tools_to_use" class="step-tools">
          <span class="tools-label">工具:</span> {{ step.tools_to_use }}
        </div>
      </div>
    </div>
    
    <div v-else class="no-steps">
      未提供具体步骤
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { PlanData } from '@/types/tool-results';

interface Props {
  data: PlanData;
}

const props = defineProps<Props>();

const getStatusEmoji = (status: string): string => {
  const emojiMap: Record<string, string> = {
    pending: '⏳',
    in_progress: '🔄',
    completed: '✅',
    skipped: '⏭️'
  };
  return emojiMap[status] || '⏳';
};

const getStatusText = (status: string): string => {
  const textMap: Record<string, string> = {
    pending: '待处理',
    in_progress: '进行中',
    completed: '已完成',
    skipped: '已跳过'
  };
  return textMap[status] || status;
};
</script>

<style lang="less" scoped>
.plan-display {
  font-size: 13px;
  color: #333;
  background: #f9fafb;
  padding: 12px;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.plan-task {
  margin-bottom: 16px;
  padding: 10px;
  background: #fff;
  border-radius: 4px;
  border-left: 3px solid #3b82f6;
  
  strong {
    color: #1f2937;
    font-weight: 600;
  }
}

.plan-steps {
  .steps-header {
    font-weight: 600;
    color: #1f2937;
    margin-bottom: 12px;
    font-size: 14px;
  }
}

.step-item {
  margin-bottom: 12px;
  padding: 10px;
  background: #fff;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
  transition: all 0.2s;
  
  &:hover {
    border-color: #d1d5db;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  }
  
  &:last-child {
    margin-bottom: 0;
  }
}

.step-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.step-number {
  font-weight: 600;
  color: #6b7280;
  min-width: 20px;
}

.step-status {
  font-size: 16px;
  line-height: 1;
}

.step-status-text {
  font-weight: 500;
  font-size: 12px;
  
  &.status-pending {
    color: #6b7280;
  }
  
  &.status-in_progress {
    color: #3b82f6;
  }
  
  &.status-completed {
    color: #10b981;
  }
  
  &.status-skipped {
    color: #9ca3af;
  }
}

.step-description {
  flex: 1;
  color: #1f2937;
  line-height: 1.5;
}

.step-tools {
  margin-top: 6px;
  padding-left: 26px;
  font-size: 12px;
  color: #6b7280;
  
  .tools-label {
    font-weight: 500;
  }
}

.no-steps {
  padding: 16px;
  text-align: center;
  color: #9ca3af;
  font-style: italic;
}
</style>

