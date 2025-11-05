<template>
  <div class="kb-chunking-settings">
    <div class="section-header">
      <h2>分块设置</h2>
      <p class="section-description">配置文档分块参数，优化检索效果</p>
    </div>

    <div class="settings-group">
      <!-- Chunk Size -->
      <div class="setting-row">
        <div class="setting-info">
          <label>分块大小</label>
          <p class="desc">控制每个文档分块的字符数（100-4000）</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localChunkSize"
              :min="100"
              :max="4000"
              :step="50"
              :marks="{ 100: '100', 1000: '1000', 2000: '2000', 4000: '4000' }"
              @change="handleChunkSizeChange"
              style="width: 200px;"
            />
            <span class="value-display">{{ localChunkSize }} 字符</span>
          </div>
        </div>
      </div>

      <!-- Chunk Overlap -->
      <div class="setting-row">
        <div class="setting-info">
          <label>分块重叠</label>
          <p class="desc">相邻文档块之间的重叠字符数（0-500）</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localChunkOverlap"
              :min="0"
              :max="500"
              :step="20"
              :marks="{ 0: '0', 250: '250', 500: '500' }"
              @change="handleChunkOverlapChange"
              style="width: 200px;"
            />
            <span class="value-display">{{ localChunkOverlap }} 字符</span>
          </div>
        </div>
      </div>

      <!-- 分隔符 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>分隔符</label>
          <p class="desc">文档分块时使用的分隔符</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localSeparators"
            :options="separatorOptions"
            multiple
            placeholder="选择分隔符"
            @change="handleSeparatorsChange"
            style="width: 280px;"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface ChunkingConfig {
  chunkSize: number
  chunkOverlap: number
  separators: string[]
}

interface Props {
  config: ChunkingConfig
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:config': [value: ChunkingConfig]
}>()

const localChunkSize = ref(props.config.chunkSize)
const localChunkOverlap = ref(props.config.chunkOverlap)
const localSeparators = ref([...props.config.separators])

// 分隔符选项
const separatorOptions = [
  { label: '双换行 (\\n\\n)', value: '\n\n' },
  { label: '单换行 (\\n)', value: '\n' },
  { label: '中文句号 (。)', value: '。' },
  { label: '感叹号 (！)', value: '！' },
  { label: '问号 (？)', value: '？' },
  { label: '中文分号 (；)', value: '；' },
  { label: '英文分号 (;)', value: ';' },
  { label: '空格 ( )', value: ' ' }
]

// 监听props变化
watch(() => props.config, (newConfig) => {
  localChunkSize.value = newConfig.chunkSize
  localChunkOverlap.value = newConfig.chunkOverlap
  localSeparators.value = [...newConfig.separators]
}, { deep: true })

// 处理分块大小变化
const handleChunkSizeChange = () => {
  emitUpdate()
}

// 处理分块重叠变化
const handleChunkOverlapChange = () => {
  emitUpdate()
}

// 处理分隔符变化
const handleSeparatorsChange = () => {
  emitUpdate()
}

// 发出更新事件
const emitUpdate = () => {
  emit('update:config', {
    chunkSize: localChunkSize.value,
    chunkOverlap: localChunkOverlap.value,
    separators: localSeparators.value
  })
}
</script>

<style lang="less" scoped>
.kb-chunking-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-border);

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.slider-container {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
  justify-content: flex-end;
}

.value-display {
  font-size: 14px;
  color: var(--td-text-color-primary);
  font-weight: 500;
  min-width: 80px;
  text-align: right;
}
</style>

