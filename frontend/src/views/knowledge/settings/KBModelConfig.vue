<template>
  <div class="kb-model-config">
    <div class="section-header">
      <h2>模型配置</h2>
      <p class="section-description">为知识库选择合适的AI模型</p>
    </div>

    <div class="settings-group">
      <!-- LLM 大语言模型 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>LLM 大语言模型 <span class="required">*</span></label>
          <p class="desc">用于对话和问答的大语言模型</p>
        </div>
        <div class="setting-control">
          <ModelSelector
            ref="llmSelectorRef"
            model-type="KnowledgeQA"
            :selected-model-id="config.llmModelId"
            :all-models="allModels"
            @update:selected-model-id="handleLLMChange"
            @add-model="handleAddModel('chat')"
            placeholder="请选择LLM模型"
          />
        </div>
      </div>

      <!-- Embedding 嵌入模型 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>Embedding 嵌入模型 <span class="required">*</span></label>
          <p class="desc">用于文本向量化的嵌入模型</p>
          <t-alert 
            v-if="hasFiles" 
            theme="warning" 
            message="知识库中已有文件，无法修改Embedding模型" 
            style="margin-top: 8px;"
          />
        </div>
        <div class="setting-control">
          <ModelSelector
            ref="embeddingSelectorRef"
            model-type="Embedding"
            :selected-model-id="config.embeddingModelId"
            :all-models="allModels"
            :disabled="hasFiles"
            @update:selected-model-id="handleEmbeddingChange"
            @add-model="handleAddModel('embedding')"
            placeholder="请选择Embedding模型"
          />
        </div>
      </div>

      <!-- ReRank 重排序模型 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>ReRank 重排序模型</label>
          <p class="desc">用于搜索结果重排序的模型（可选）</p>
        </div>
        <div class="setting-control">
          <ModelSelector
            ref="rerankSelectorRef"
            model-type="Rerank"
            :selected-model-id="config.rerankModelId"
            :all-models="allModels"
            @update:selected-model-id="handleRerankChange"
            @add-model="handleAddModel('rerank')"
            placeholder="请选择ReRank模型（可选）"
          />
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useUIStore } from '@/stores/ui'
import ModelSelector from '@/components/ModelSelector.vue'

interface ModelConfig {
  llmModelId?: string
  embeddingModelId?: string
  rerankModelId?: string
  vllmModelId?: string
}

interface Props {
  config: ModelConfig
  hasFiles: boolean
  allModels?: any[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:config': [value: ModelConfig]
}>()

const uiStore = useUIStore()

// 引用各个模型选择器
const llmSelectorRef = ref<InstanceType<typeof ModelSelector>>()
const embeddingSelectorRef = ref<InstanceType<typeof ModelSelector>>()
const rerankSelectorRef = ref<InstanceType<typeof ModelSelector>>()

// 处理LLM模型变化
const handleLLMChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    llmModelId: modelId
  })
}

// 处理Embedding模型变化
const handleEmbeddingChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    embeddingModelId: modelId
  })
}

// 处理ReRank模型变化
const handleRerankChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    rerankModelId: modelId
  })
}

// 处理添加模型按钮点击
const handleAddModel = (subSection: string) => {
  // 打开全局设置对话框，并导航到对应的模型子页面
  uiStore.openSettings('models', subSection)
}

// 由于使用了 allModels prop，不再需要单独刷新各个选择器
</script>

<style lang="less" scoped>
.kb-model-config {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #333333;
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: #666666;
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
  border-bottom: 1px solid #e5e7eb;

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
    color: #333333;
    display: block;
    margin-bottom: 4px;

    .required {
      color: #fa5151;
      margin-left: 2px;
    }
  }

  .desc {
    font-size: 13px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: flex-start;
}
</style>

