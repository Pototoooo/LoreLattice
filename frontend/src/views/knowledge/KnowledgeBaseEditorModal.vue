<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="settings-overlay" @click.self="handleClose">
        <div class="settings-modal">
          <!-- 关闭按钮 -->
          <button class="close-btn" @click="handleClose" aria-label="关闭">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>

          <div class="settings-container">
            <!-- 左侧导航 -->
            <div class="settings-sidebar">
              <div class="sidebar-header">
                <h2 class="sidebar-title">{{ mode === 'create' ? '新建知识库' : '知识库设置' }}</h2>
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
                  <div v-if="formData" class="section-content">
                    <div class="section-header">
                      <h3 class="section-title">基本信息</h3>
                      <p class="section-desc">设置知识库的名称和描述信息</p>
                    </div>
                    <div class="section-body">
                      <div class="form-item">
                        <label class="form-label required">知识库名称</label>
                        <t-input 
                          v-model="formData.name" 
                          placeholder="请输入知识库名称"
                          :maxlength="50"
                        />
                      </div>
                      <div class="form-item">
                        <label class="form-label">知识库描述</label>
                        <t-textarea 
                          v-model="formData.description" 
                          placeholder="请输入知识库描述（可选）"
                          :maxlength="200"
                          :autosize="{ minRows: 3, maxRows: 6 }"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 模型配置 -->
                <div v-show="currentSection === 'models'" class="section">
                  <KBModelConfig
                    ref="modelConfigRef"
                    v-if="formData"
                    :config="formData.modelConfig"
                    :has-files="hasFiles"
                    :all-models="allModels"
                    @update:config="handleModelConfigUpdate"
                  />
                </div>

                <!-- 分块设置 -->
                <div v-show="currentSection === 'chunking'" class="section">
                  <KBChunkingSettings
                    v-if="formData"
                    :config="formData.chunkingConfig"
                    @update:config="handleChunkingConfigUpdate"
                  />
                </div>

                <!-- 高级设置 -->
                <div v-show="currentSection === 'advanced'" class="section">
                  <KBAdvancedSettings
                    ref="advancedSettingsRef"
                    v-if="formData"
                    :multimodal="formData.multimodalConfig"
                    :node-extract="formData.nodeExtractConfig"
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
                <t-button theme="primary" @click="handleSubmit" :loading="saving">
                  {{ mode === 'create' ? '创建知识库' : '保存配置' }}
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
import { ref, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { createKnowledgeBase, getKnowledgeBaseById, listKnowledgeFiles, updateKnowledgeBase } from '@/api/knowledge-base'
import { updateKBConfig, type KBModelConfigRequest } from '@/api/initialization'
import { listModels } from '@/api/model'
import KBModelConfig from './settings/KBModelConfig.vue'
import KBChunkingSettings from './settings/KBChunkingSettings.vue'
import KBAdvancedSettings from './settings/KBAdvancedSettings.vue'

// Props
const props = defineProps<{
  visible: boolean
  mode: 'create' | 'edit'
  kbId?: string
}>()

// Emits
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success', kbId: string): void
}>()

const currentSection = ref<string>('basic')
const saving = ref(false)
const loading = ref(false)
const allModels = ref<any[]>([])
const hasFiles = ref(false)

const navItems = [
  { key: 'basic', icon: 'ℹ️', label: '基本信息' },
  { key: 'models', icon: '🤖', label: '模型配置' },
  { key: 'chunking', icon: '📄', label: '分块设置' },
  { key: 'advanced', icon: '⚙️', label: '高级设置' }
]

// 模型配置引用
const modelConfigRef = ref<InstanceType<typeof KBModelConfig>>()
const advancedSettingsRef = ref<InstanceType<typeof KBAdvancedSettings>>()

// 表单数据
const formData = ref<any>(null)

// 初始化表单数据
const initFormData = () => {
  return {
    name: '',
    description: '',
    modelConfig: {
      llmModelId: '',
      embeddingModelId: '',
      rerankModelId: ''
    },
    chunkingConfig: {
      chunkSize: 512,
      chunkOverlap: 100,
      separators: ['\n\n', '\n', '。', '！', '？', ';', '；']
    },
    multimodalConfig: {
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
    },
    nodeExtractConfig: {
      enabled: false,
      text: '',
      tags: [] as string[]
    }
  }
}

// 加载所有模型
const loadAllModels = async () => {
  try {
    const models = await listModels()
    allModels.value = models || []
  } catch (error) {
    console.error('加载模型列表失败:', error)
    MessagePlugin.error('加载模型列表失败')
    allModels.value = []
  }
}

// 加载知识库数据（编辑模式）
const loadKBData = async () => {
  if (props.mode !== 'edit' || !props.kbId) return
  
  loading.value = true
  try {
    const [kbInfo, models, filesResult] = await Promise.all([
      getKnowledgeBaseById(props.kbId),
      loadAllModels(),
      listKnowledgeFiles(props.kbId, { page: 1, page_size: 1 })
    ])
    
    if (!kbInfo || !kbInfo.data) {
      throw new Error('知识库不存在')
    }

    const kb = kbInfo.data
    hasFiles.value = (filesResult as any)?.total > 0
    
    // 设置表单数据
    formData.value = {
      name: kb.name || '',
      description: kb.description || '',
      modelConfig: {
        llmModelId: kb.summary_model_id || '',
        embeddingModelId: kb.embedding_model_id || '',
        rerankModelId: kb.rerank_model_id || ''
      },
      chunkingConfig: {
        chunkSize: kb.chunking_config?.chunk_size || 512,
        chunkOverlap: kb.chunking_config?.chunk_overlap || 100,
        separators: kb.chunking_config?.separators || ['\n\n', '\n', '。', '！', '？', ';', '；']
      },
      multimodalConfig: {
        enabled: !!(kb.vlm_model_id || (kb.cos_config?.provider && kb.cos_config?.bucket_name)),
        storageType: (kb.cos_config?.provider || 'minio') as 'minio' | 'cos',
        vllmModelId: kb.vlm_model_id || '',
        minio: {
          bucketName: kb.cos_config?.bucket_name || '',
          useSSL: kb.cos_config?.use_ssl || false,
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
      },
      nodeExtractConfig: {
        enabled: kb.extract_config?.enabled || false,
        text: kb.extract_config?.text || '',
        tags: kb.extract_config?.tags || []
      }
    }
  } catch (error) {
    console.error('加载知识库数据失败:', error)
    MessagePlugin.error('加载知识库数据失败')
    handleClose()
  } finally {
    loading.value = false
  }
}

// 处理配置更新
const handleModelConfigUpdate = (config: any) => {
  if (formData.value) {
    formData.value.modelConfig = { ...config }
  }
}

const handleChunkingConfigUpdate = (config: any) => {
  if (formData.value) {
    formData.value.chunkingConfig = { ...config }
  }
}

const handleMultimodalUpdate = (config: any) => {
  if (formData.value) {
    formData.value.multimodalConfig = { ...config }
  }
}

const handleNodeExtractUpdate = (config: any) => {
  if (formData.value) {
    formData.value.nodeExtractConfig = { ...config }
  }
}

// 验证表单
const validateForm = (): boolean => {
  if (!formData.value) return false

  // 验证基本信息
  if (!formData.value.name || !formData.value.name.trim()) {
    MessagePlugin.warning('请输入知识库名称')
    currentSection.value = 'basic'
    return false
  }

  // 验证模型配置 - 必须配置 embedding 和 summary 模型
  if (!formData.value.modelConfig.embeddingModelId) {
    MessagePlugin.warning('请选择 Embedding 模型')
    currentSection.value = 'models'
    return false
  }

  if (!formData.value.modelConfig.llmModelId) {
    MessagePlugin.warning('请选择 Summary 模型')
    currentSection.value = 'models'
    return false
  }

  // 验证多模态配置（如果启用）
  if (formData.value.multimodalConfig.enabled) {
    const validation = advancedSettingsRef.value?.validateMultimodal()
    if (validation && !validation.valid) {
      MessagePlugin.warning(validation.message || '多模态配置验证失败')
      currentSection.value = 'advanced'
      return false
    }
  }

  return true
}

// 构建提交数据
const buildSubmitData = () => {
  if (!formData.value) return null

  const data: any = {
    name: formData.value.name,
    description: formData.value.description,
    chunking_config: {
      chunk_size: formData.value.chunkingConfig.chunkSize,
      chunk_overlap: formData.value.chunkingConfig.chunkOverlap,
      separators: formData.value.chunkingConfig.separators,
      enable_multimodal: formData.value.multimodalConfig.enabled
    },
    embedding_model_id: formData.value.modelConfig.embeddingModelId,
    summary_model_id: formData.value.modelConfig.llmModelId
  }

  // 可选的 Rerank 模型
  if (formData.value.modelConfig.rerankModelId) {
    data.rerank_model_id = formData.value.modelConfig.rerankModelId
  }

  // 添加多模态配置
  if (formData.value.multimodalConfig.enabled) {
    if (formData.value.multimodalConfig.vllmModelId) {
      data.vlm_model_id = formData.value.multimodalConfig.vllmModelId
    }
    
    const storageType = formData.value.multimodalConfig.storageType
    if (storageType === 'minio') {
      data.cos_config = {
        provider: 'minio',
        bucket_name: formData.value.multimodalConfig.minio.bucketName,
        use_ssl: formData.value.multimodalConfig.minio.useSSL,
        path_prefix: formData.value.multimodalConfig.minio.pathPrefix || undefined
      }
    } else {
      data.cos_config = {
        provider: 'cos',
        secret_id: formData.value.multimodalConfig.cos.secretId,
        secret_key: formData.value.multimodalConfig.cos.secretKey,
        region: formData.value.multimodalConfig.cos.region,
        bucket_name: formData.value.multimodalConfig.cos.bucketName,
        app_id: formData.value.multimodalConfig.cos.appId,
        path_prefix: formData.value.multimodalConfig.cos.pathPrefix || undefined
      }
    }
  }

  // 添加知识图谱配置
  if (formData.value.nodeExtractConfig.enabled) {
    data.extract_config = {
      enabled: true,
      text: formData.value.nodeExtractConfig.text,
      tags: formData.value.nodeExtractConfig.tags
    }
  }

  return data
}

// 提交表单
const handleSubmit = async () => {
  if (!validateForm()) {
    return
  }

  saving.value = true
  try {
    const data = buildSubmitData()
    if (!data) {
      throw new Error('数据构建失败')
    }

    if (props.mode === 'create') {
      // 创建模式：一次性创建知识库及所有配置
      const result = await createKnowledgeBase(data)
      if (!result.success || !result.data?.id) {
        throw new Error(result.message || '创建知识库失败')
      }
      MessagePlugin.success('知识库创建成功')
      emit('success', result.data.id)
    } else {
      // 编辑模式：分别更新基本信息和配置
      if (!props.kbId) {
        throw new Error('缺少知识库ID')
      }

      // 1. 更新基本信息（名称、描述）
      await updateKnowledgeBase(props.kbId, {
        name: data.name,
        description: data.description,
        config: {} // 空配置，只更新基本信息
      })

      // 2. 更新完整配置（模型、分块、多模态、知识图谱等）
      const config: KBModelConfigRequest = {
        llmModelId: data.summary_model_id,
        embeddingModelId: data.embedding_model_id,
        rerankModelId: data.rerank_model_id || '',
        vllmModelId: data.vlm_model_id || '',
        documentSplitting: {
          chunkSize: data.chunking_config.chunk_size,
          chunkOverlap: data.chunking_config.chunk_overlap,
          separators: data.chunking_config.separators
        },
        multimodal: {
          enabled: !!data.cos_config,
          storageType: data.cos_config?.provider || 'minio',
          cos: data.cos_config?.provider === 'cos' ? {
            secretId: data.cos_config.secret_id,
            secretKey: data.cos_config.secret_key,
            region: data.cos_config.region,
            bucketName: data.cos_config.bucket_name,
            appId: data.cos_config.app_id,
            pathPrefix: data.cos_config.path_prefix || ''
          } : undefined,
          minio: data.cos_config?.provider === 'minio' ? {
            bucketName: data.cos_config.bucket_name,
            useSSL: data.cos_config.use_ssl || false,
            pathPrefix: data.cos_config.path_prefix || ''
          } : undefined
        },
        nodeExtract: {
          enabled: data.extract_config?.enabled || false,
          text: data.extract_config?.text || '',
          tags: data.extract_config?.tags || [],
          nodes: [],
          relations: []
        }
      }

      await updateKBConfig(props.kbId, config)
      MessagePlugin.success('配置保存成功')
      emit('success', props.kbId)
    }
    
    handleClose()
  } catch (error: any) {
    console.error('操作失败:', error)
    MessagePlugin.error(error.message || '操作失败')
  } finally {
    saving.value = false
  }
}

// 重置所有状态
const resetState = () => {
  currentSection.value = 'basic'
  formData.value = null
  hasFiles.value = false
  saving.value = false
  loading.value = false
}

// 关闭弹窗
const handleClose = () => {
  emit('update:visible', false)
  setTimeout(() => {
    resetState()
  }, 300)
}

// 监听弹窗打开/关闭
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    // 打开弹窗时，先重置状态
    resetState()
    
    // 加载模型列表
    await loadAllModels()
    
    // 根据模式加载数据
    if (props.mode === 'edit' && props.kbId) {
      await loadKBData()
    } else {
      // 创建模式：初始化空表单
      formData.value = initFormData()
      hasFiles.value = false
    }
  } else {
    // 关闭弹窗时，延迟重置状态（等待动画结束）
    setTimeout(() => {
      resetState()
    }, 300)
  }
})
</script>

<style scoped lang="less">
// 复用创建知识库的样式
.settings-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.settings-modal {
  position: relative;
  width: 90vw;
  max-width: 1100px;
  height: 85vh;
  max-height: 750px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.close-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 32px;
  height: 32px;
  border: none;
  background: #f5f5f5;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
  transition: all 0.2s ease;
  z-index: 10;

  &:hover {
    background: #e5e5e5;
    color: #000;
  }
}

.settings-container {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.settings-sidebar {
  width: 200px;
  background: #fafafa;
  border-right: 1px solid #e5e5e5;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 24px 20px;
  border-bottom: 1px solid #e5e5e5;
}

.sidebar-title {
  margin: 0;
  font-family: "PingFang SC";
  font-size: 18px;
  font-weight: 600;
  color: #000000e6;
}

.settings-nav {
  flex: 1;
  padding: 12px 8px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  margin-bottom: 4px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: "PingFang SC";
  font-size: 14px;
  color: #00000099;

  &:hover {
    background: #f0f0f0;
  }

  &.active {
    background: #07c05f1a;
    color: #07c05f;
    font-weight: 500;
  }
}

.nav-icon {
  margin-right: 8px;
  font-size: 16px;
}

.nav-label {
  flex: 1;
}

.settings-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content-wrapper {
  flex: 1;
  overflow-y: auto;
  padding: 24px 32px;
}

.section {
  margin-bottom: 32px;

  &:last-child {
    margin-bottom: 0;
  }
}

.section-content {
  .section-header {
    margin-bottom: 20px;
  }

  .section-title {
    margin: 0 0 8px 0;
    font-family: "PingFang SC";
    font-size: 16px;
    font-weight: 600;
    color: #000000e6;
  }

  .section-desc {
    margin: 0;
    font-family: "PingFang SC";
    font-size: 14px;
    color: #00000066;
    line-height: 22px;
  }

  .section-body {
    background: #fff;
  }
}

.form-item {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-family: "PingFang SC";
  font-size: 14px;
  font-weight: 500;
  color: #000000e6;

  &.required::after {
    content: '*';
    color: #FA5151;
    margin-left: 4px;
  }
}

.settings-footer {
  padding: 16px 32px;
  border-top: 1px solid #e5e5e5;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

// 过渡动画
.modal-enter-active,
.modal-leave-active {
  transition: all 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;

  .settings-modal {
    transform: scale(0.95);
  }
}
</style>

