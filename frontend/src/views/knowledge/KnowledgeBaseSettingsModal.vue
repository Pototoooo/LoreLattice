<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="settings-overlay" @click.self="handleClose">
        <div class="settings-modal">
          <!-- 关闭按钮 -->
          <button class="close-btn" @click="handleClose" aria-label="关闭设置">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>

          <div class="settings-container">
            <!-- 左侧导航 -->
            <div class="settings-sidebar">
              <div class="sidebar-header">
                <h2 class="sidebar-title">知识库设置</h2>
              </div>
              <div class="settings-nav">
                <div 
                  v-for="(item, index) in navItems" 
                  :key="index"
                  :class="['nav-item', { 'active': currentSection === item.key }]"
                  @click="currentSection = item.key"
                >
                  <span class="nav-icon">{{ item.icon }}</span>
                  <span class="nav-label">{{ item.label }}</span>
                </div>
              </div>
            </div>

            <!-- 右侧内容区域 -->
            <div class="settings-content">
              <div class="content-wrapper">
                <!-- 基本信息 -->
                <div v-show="currentSection === 'basic'" class="section">
                  <KBBasicInfo
                    v-if="kbData"
                    :kb-id="kbId"
                    :name="kbData.name"
                    :description="kbData.description"
                    @update:name="kbData.name = $event"
                    @update:description="kbData.description = $event"
                  />
                </div>

                <!-- 模型配置 -->
                <div v-show="currentSection === 'models'" class="section">
                  <KBModelConfig
                    ref="modelConfigRef"
                    v-if="configData"
                    :config="modelConfig"
                    :has-files="configData.hasFiles"
                    :all-models="allModels"
                    @update:config="handleModelConfigUpdate"
                  />
                </div>

                <!-- 分块设置 -->
                <div v-show="currentSection === 'chunking'" class="section">
                  <KBChunkingSettings
                    v-if="configData"
                    :config="chunkingConfig"
                    @update:config="handleChunkingConfigUpdate"
                  />
                </div>

                <!-- 高级设置 -->
                <div v-show="currentSection === 'advanced'" class="section">
                  <KBAdvancedSettings
                    ref="advancedSettingsRef"
                    v-if="configData"
                    :multimodal="multimodalConfig"
                    :node-extract="nodeExtractConfig"
                    :all-models="allModels"
                    @update:multimodal="handleMultimodalUpdate"
                    @update:nodeExtract="handleNodeExtractUpdate"
                  />
                </div>
              </div>

              <!-- 保存按钮 -->
              <div class="settings-footer">
                <t-button theme="default" variant="outline" @click="handleClose">
                  取消
                </t-button>
                <t-button theme="primary" @click="handleSave" :loading="saving">
                  保存配置
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useUIStore } from '@/stores/ui'
import { MessagePlugin } from 'tdesign-vue-next'
import { getKnowledgeBaseById, listKnowledgeFiles } from '@/api/knowledge-base'
import { updateKBConfig, type KBModelConfigRequest } from '@/api/initialization'
import { listModels } from '@/api/model'
import KBBasicInfo from './settings/KBBasicInfo.vue'
import KBModelConfig from './settings/KBModelConfig.vue'
import KBChunkingSettings from './settings/KBChunkingSettings.vue'
import KBAdvancedSettings from './settings/KBAdvancedSettings.vue'

const uiStore = useUIStore()

const visible = computed(() => uiStore.showKBSettingsModal)
const kbId = computed(() => uiStore.currentKBId || '')

const currentSection = ref<string>('basic')
const saving = ref(false)
const loading = ref(false)
const allModels = ref<any[]>([])

const navItems = [
  { key: 'basic', icon: 'ℹ️', label: '基本信息' },
  { key: 'models', icon: '🤖', label: '模型配置' },
  { key: 'chunking', icon: '📄', label: '分块设置' },
  { key: 'advanced', icon: '⚙️', label: '高级设置' }
]

// 知识库基本数据
const kbData = ref<{
  name: string
  description?: string
} | null>(null)

// 配置数据（简化为只需要 hasFiles 标记）
const configData = ref<{ hasFiles: boolean } | null>(null)

// 模型配置引用
const modelConfigRef = ref<InstanceType<typeof KBModelConfig>>()
const advancedSettingsRef = ref<InstanceType<typeof KBAdvancedSettings>>()

// 模型配置
const modelConfig = ref({
  llmModelId: '',
  embeddingModelId: '',
  rerankModelId: ''
})

// 分块配置
const chunkingConfig = ref({
  chunkSize: 512,
  chunkOverlap: 100,
  separators: ['\n\n', '\n', '。', '！', '？', ';', '；']
})

// 多模态配置
const multimodalConfig = ref({
  enabled: false,
  storageType: 'minio' as 'minio' | 'cos',
  vllmModelId: '',
  minio: {
    bucketName: '',
    useSSL: false,
    pathPrefix: ''
  },
  cos: {
    secretId: '',
    secretKey: '',
    region: '',
    bucketName: '',
    appId: '',
    pathPrefix: ''
  }
})

// 知识图谱配置
const nodeExtractConfig = ref({
  enabled: false,
  text: '',
  tags: [] as string[]
})

// 加载所有模型（统一加载，避免重复调用）
const loadAllModels = async () => {
  try {
    const models = await listModels()
    allModels.value = models || []
    return models || []
  } catch (error) {
    console.error('加载模型列表失败:', error)
    MessagePlugin.error('加载模型列表失败')
    allModels.value = []
    return []
  }
}

// 加载知识库数据
const loadKBData = async () => {
  if (!kbId.value) return
  
  loading.value = true
  try {
    // 并行加载知识库信息和所有模型（只需要一个接口！）
    const [kbInfo, models] = await Promise.all([
      getKnowledgeBaseById(kbId.value),
      loadAllModels()
    ])
    
    if (!kbInfo || !kbInfo.data) {
      throw new Error('知识库不存在')
    }

    const kb = kbInfo.data
    
    // 设置基本信息
    kbData.value = {
      name: kb.name || '',
      description: kb.description || ''
    }

    // 设置模型ID（直接从知识库数据中获取）
    modelConfig.value = {
      llmModelId: kb.summary_model_id || '',
      embeddingModelId: kb.embedding_model_id || '',
      rerankModelId: kb.rerank_model_id || ''
    }

    // 设置分块配置
    chunkingConfig.value = {
      chunkSize: kb.chunking_config?.chunk_size || 512,
      chunkOverlap: kb.chunking_config?.chunk_overlap || 100,
      separators: kb.chunking_config?.separators || ['\n\n', '\n', '。', '！', '？', ';', '；']
    }

    // 设置多模态配置
    // 判断是否真的有存储配置：provider 不为空且有实际的存储信息
    const hasStorage = kb.cos_config?.provider && (
      (kb.cos_config.provider === 'minio' && kb.cos_config.bucket_name) ||
      (kb.cos_config.provider === 'cos' && kb.cos_config.bucket_name)
    )
    const hasVLM = !!kb.vlm_model_id
    
    multimodalConfig.value = {
      enabled: !!(hasStorage || hasVLM),
      storageType: (kb.cos_config?.provider || 'minio') as 'minio' | 'cos',
      vllmModelId: kb.vlm_model_id || '',
      minio: {
        bucketName: kb.cos_config?.bucket_name || '',
        useSSL: false,
        pathPrefix: kb.cos_config?.path_prefix || ''
      },
      cos: {
        secretId: kb.cos_config?.secret_id || '',
        secretKey: kb.cos_config?.secret_key || '',
        region: kb.cos_config?.region || '',
        bucketName: kb.cos_config?.bucket_name || '',
        appId: kb.cos_config?.app_id || '',
        pathPrefix: kb.cos_config?.path_prefix || ''
      }
    }

    // 设置知识图谱配置
    if (kb.extract_config) {
      nodeExtractConfig.value = {
        enabled: true,
        text: kb.extract_config.text || '',
        tags: kb.extract_config.tags || []
      }
    } else {
      nodeExtractConfig.value = {
        enabled: false,
        text: '',
        tags: []
      }
    }

    // 检查是否有文件（用于判断是否可以修改Embedding模型）
    const knowledgeList = await listKnowledgeFiles(kbId.value, { page: 1, page_size: 1 })
    const hasFiles = knowledgeList && knowledgeList.data && knowledgeList.data.items && knowledgeList.data.items.length > 0
    
    // 设置 configData（用于传递 hasFiles 状态）
    configData.value = { hasFiles } as any

  } catch (error) {
    console.error('加载知识库数据失败:', error)
    MessagePlugin.error('加载知识库数据失败')
  } finally {
    loading.value = false
  }
}

// 不再需要 mapModelIds，直接使用知识库返回的模型ID

// 处理模型配置更新
const handleModelConfigUpdate = (newConfig: typeof modelConfig.value) => {
  modelConfig.value = newConfig
}

// 处理分块配置更新
const handleChunkingConfigUpdate = (newConfig: typeof chunkingConfig.value) => {
  chunkingConfig.value = newConfig
}

// 处理多模态配置更新
const handleMultimodalUpdate = (newConfig: typeof multimodalConfig.value) => {
  multimodalConfig.value = newConfig
}

// 处理知识图谱配置更新
const handleNodeExtractUpdate = (newConfig: typeof nodeExtractConfig.value) => {
  nodeExtractConfig.value = newConfig
}

// 保存配置
const handleSave = async () => {
  if (saving.value || !kbId.value) return

  // 验证必填项 - 模型配置
  if (!modelConfig.value.llmModelId) {
    MessagePlugin.warning('请选择LLM模型')
    currentSection.value = 'models'
    return
  }

  if (!modelConfig.value.embeddingModelId) {
    MessagePlugin.warning('请选择Embedding模型')
    currentSection.value = 'models'
    return
  }

  // 验证多模态配置
  if (multimodalConfig.value.enabled) {
    // 验证VLLM模型（必选）
    if (!multimodalConfig.value.vllmModelId) {
      MessagePlugin.warning('多模态功能已启用，请选择VLLM视觉模型')
      currentSection.value = 'advanced'
      return
    }

    // 验证存储配置
    if (multimodalConfig.value.storageType === 'minio') {
      if (!multimodalConfig.value.minio.bucketName) {
        MessagePlugin.warning('使用MinIO存储时，Bucket名称为必填项')
        currentSection.value = 'advanced'
        return
      }
    } else if (multimodalConfig.value.storageType === 'cos') {
      const cos = multimodalConfig.value.cos
      if (!cos.secretId) {
        MessagePlugin.warning('使用腾讯云COS时，SecretId为必填项')
        currentSection.value = 'advanced'
        return
      }
      if (!cos.secretKey) {
        MessagePlugin.warning('使用腾讯云COS时，SecretKey为必填项')
        currentSection.value = 'advanced'
        return
      }
      if (!cos.region) {
        MessagePlugin.warning('使用腾讯云COS时，地域为必填项')
        currentSection.value = 'advanced'
        return
      }
      if (!cos.bucketName) {
        MessagePlugin.warning('使用腾讯云COS时，Bucket名称为必填项')
        currentSection.value = 'advanced'
        return
      }
      if (!cos.appId) {
        MessagePlugin.warning('使用腾讯云COS时，AppId为必填项')
        currentSection.value = 'advanced'
        return
      }
    }
  }

  saving.value = true
  try {
    // 构建配置对象（简化版，只传模型ID）
    const config: KBModelConfigRequest = {
      llmModelId: modelConfig.value.llmModelId,
      embeddingModelId: modelConfig.value.embeddingModelId,
      rerankModelId: modelConfig.value.rerankModelId || '',
      vllmModelId: multimodalConfig.value.vllmModelId || '',
      documentSplitting: {
        chunkSize: chunkingConfig.value.chunkSize,
        chunkOverlap: chunkingConfig.value.chunkOverlap,
        separators: chunkingConfig.value.separators
      },
      multimodal: {
        enabled: multimodalConfig.value.enabled,
        storageType: multimodalConfig.value.enabled ? multimodalConfig.value.storageType : 'minio',
        cos: multimodalConfig.value.enabled && multimodalConfig.value.storageType === 'cos' ? multimodalConfig.value.cos : undefined,
        minio: multimodalConfig.value.enabled && multimodalConfig.value.storageType === 'minio' ? multimodalConfig.value.minio : undefined
      },
      nodeExtract: {
        enabled: nodeExtractConfig.value.enabled,
        text: nodeExtractConfig.value.text,
        tags: nodeExtractConfig.value.tags,
        nodes: [],
        relations: []
      }
    }

    // 保存配置
    await updateKBConfig(kbId.value, config)
    MessagePlugin.success('配置保存成功')
    handleClose()
  } catch (error) {
    console.error('保存配置失败:', error)
    MessagePlugin.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

// 处理关闭
const handleClose = () => {
  uiStore.closeKBSettings()
  // 重置当前选中的section
  setTimeout(() => {
    currentSection.value = 'basic'
  }, 300)
}

// 监听visible变化，当打开时加载数据
watch(visible, (newVal) => {
  if (newVal && kbId.value) {
    loadKBData()
  }
})

// 监听全局设置关闭事件，刷新模型列表
watch(() => uiStore.showSettingsModal, (newVal, oldVal) => {
  // 当全局设置从打开变为关闭，且知识库设置仍然打开时，刷新模型列表
  if (oldVal && !newVal && visible.value) {
    // 统一刷新所有模型列表（只调用一次API）
    loadAllModels()
  }
})

onMounted(() => {
  if (visible.value && kbId.value) {
    loadKBData()
  }
})
</script>

<style lang="less" scoped>
/* 遮罩层 */
.settings-overlay {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

/* 弹窗容器 */
.settings-modal {
  position: relative;
  width: 100%;
  max-width: 900px;
  height: 700px;
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 6px 28px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 关闭按钮 */
.close-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: #666666;
  cursor: pointer;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  z-index: 10;

  &:hover {
    background: #f5f5f5;
    color: #333333;
  }
}

.settings-container {
  display: flex;
  height: 100%;
  width: 100%;
  overflow: hidden;
}

/* 左侧导航栏 */
.settings-sidebar {
  width: 220px;
  background-color: #f8f9fa;
  border-right: 1px solid #e5e7eb;
  flex-shrink: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 24px 16px 16px;
  border-bottom: 1px solid #e5e7eb;
}

.sidebar-title {
  font-size: 18px;
  font-weight: 600;
  color: #333333;
  margin: 0;
}

.settings-nav {
  padding: 16px 8px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  margin-bottom: 4px;
  border-radius: 6px;
  cursor: pointer;
  color: #666666;
  font-size: 14px;
  transition: all 0.2s ease;
  user-select: none;

  &:hover {
    background-color: #e8f5ed;
    color: #333333;
  }

  &.active {
    background-color: rgba(7, 192, 95, 0.1);
    color: #07C05F;
    font-weight: 500;
  }
}

.nav-icon {
  margin-right: 12px;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.nav-label {
  flex: 1;
}

/* 右侧内容区域 */
.settings-content {
  flex: 1;
  overflow-y: auto;
  background-color: #ffffff;
  display: flex;
  flex-direction: column;
}

.content-wrapper {
  flex: 1;
  max-width: 600px;
  padding: 40px 48px;
  overflow-y: auto;
}

.section {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 底部保存按钮 */
.settings-footer {
  padding: 16px 48px;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background: #ffffff;
}

/* 弹窗动画 */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .settings-modal,
.modal-leave-active .settings-modal {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .settings-modal,
.modal-leave-to .settings-modal {
  transform: scale(0.95);
  opacity: 0;
}

/* 滚动条样式 */
.settings-sidebar::-webkit-scrollbar,
.settings-content::-webkit-scrollbar,
.content-wrapper::-webkit-scrollbar {
  width: 6px;
}

.settings-sidebar::-webkit-scrollbar-track {
  background: #f8f9fa;
}

.settings-sidebar::-webkit-scrollbar-thumb {
  background: #d0d0d0;
  border-radius: 3px;
}

.settings-sidebar::-webkit-scrollbar-thumb:hover {
  background: #b0b0b0;
}

.content-wrapper::-webkit-scrollbar-track {
  background: #ffffff;
}

.content-wrapper::-webkit-scrollbar-thumb {
  background: #d0d0d0;
  border-radius: 3px;
}

.content-wrapper::-webkit-scrollbar-thumb:hover {
  background: #b0b0b0;
}
</style>

