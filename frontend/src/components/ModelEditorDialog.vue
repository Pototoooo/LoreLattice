<template>
  <t-dialog
    v-model:visible="dialogVisible"
    :header="isEdit ? '编辑模型' : '添加模型'"
    :width="600"
    :on-confirm="handleConfirm"
    :on-cancel="handleCancel"
    :confirm-btn="{ content: '保存', loading: saving }"
  >
    <div class="model-editor-form">
      <t-form ref="formRef" :data="formData" :rules="rules" layout="vertical">
        <!-- 模型名称 -->
        <t-form-item label="模型名称" name="name">
          <t-input v-model="formData.name" placeholder="为模型取个名字，如：GPT-4" />
        </t-form-item>

        <!-- 模型来源 -->
        <t-form-item label="模型来源" name="source">
          <t-radio-group v-model="formData.source">
            <t-radio value="local">Ollama (本地)</t-radio>
            <t-radio value="remote">Remote API (远程)</t-radio>
          </t-radio-group>
        </t-form-item>

        <!-- VLLM 专用：接口类型（在来源选择之后） -->
        <template v-if="modelType === 'vllm' && formData.source === 'local'">
          <t-form-item label="接口类型" name="interfaceType">
            <t-radio-group v-model="formData.interfaceType">
              <t-radio value="ollama">Ollama</t-radio>
              <t-radio value="openai">OpenAI 兼容接口</t-radio>
            </t-radio-group>
          </t-form-item>
        </template>

        <!-- Ollama 本地模型选择器 -->
        <t-form-item v-if="formData.source === 'local' && modelType !== 'vllm'" label="模型标识" name="modelName">
          <t-select
            v-model="formData.modelName"
            :loading="loadingOllamaModels"
            :class="{ 'downloading': downloading }"
            :style="downloading ? `--progress: ${downloadProgress}%` : ''"
            filterable
            :filter="handleModelFilter"
            placeholder="搜索模型..."
            @focus="loadOllamaModels"
            @visible-change="handleDropdownVisibleChange"
          >
            <!-- 已下载的模型 -->
            <t-option
              v-for="model in filteredOllamaModels"
              :key="model.name"
              :value="model.name"
              :label="model.name"
            >
              <div class="model-option">
                <t-icon name="check-circle-filled" class="downloaded-icon" />
                <span class="model-name">{{ model.name }}</span>
                <span class="model-size">{{ formatModelSize(model.size) }}</span>
              </div>
            </t-option>
            
            <!-- 下载新模型选项（仅当搜索词不在列表中时显示） -->
            <t-option
              v-if="showDownloadOption"
              :value="`__download__${searchKeyword}`"
              :label="`下载: ${searchKeyword}`"
              class="download-option"
            >
              <div class="model-option download">
                <t-icon name="download" class="download-icon" />
                <span class="model-name">下载: {{ searchKeyword }}</span>
              </div>
            </t-option>
            
            <!-- 下载进度后缀 -->
            <template v-if="downloading" #suffix>
              <div class="download-suffix">
                <t-icon name="loading" class="spinning" />
                <span class="progress-text">{{ downloadProgress.toFixed(1) }}%</span>
              </div>
            </template>
          </t-select>
          
          <!-- 刷新按钮 -->
          <t-button
            variant="text"
            size="small"
            :loading="loadingOllamaModels"
            @click="refreshOllamaModels"
            class="refresh-btn"
          >
            <t-icon name="refresh" />
            刷新列表
          </t-button>
        </t-form-item>

        <!-- Remote API 和 VLLM 保持原有的 input -->
        <t-form-item v-else label="模型标识" name="modelName">
          <t-input 
            v-model="formData.modelName" 
            :placeholder="formData.source === 'local' ? '如：llava:latest' : '如：gpt-4, claude-3-opus'"
          />
        </t-form-item>

        <!-- Remote API 配置或 VLLM OpenAI 兼容接口 -->
        <template v-if="formData.source === 'remote' || (modelType === 'vllm' && formData.interfaceType === 'openai')">
          <t-form-item label="Base URL" name="baseUrl">
            <t-input 
              v-model="formData.baseUrl" 
              :placeholder="modelType === 'vllm' ? '如：http://localhost:11434/v1' : '如：https://api.openai.com/v1'"
            />
          </t-form-item>

          <t-form-item label="API Key (可选)" name="apiKey">
            <t-input 
              v-model="formData.apiKey" 
              type="password"
              placeholder="输入 API Key"
            />
          </t-form-item>

          <!-- Remote API 校验 -->
          <t-form-item label="连接测试">
            <div class="api-test-section">
              <t-button 
                variant="outline" 
                @click="checkRemoteAPI"
                :loading="checking"
                :disabled="!formData.modelName || !formData.baseUrl"
              >
                <template #icon>
                  <t-icon 
                    v-if="!checking && remoteChecked && remoteAvailable"
                    name="check-circle-filled" 
                    class="status-icon available"
                  />
                  <t-icon 
                    v-else-if="!checking && remoteChecked && !remoteAvailable"
                    name="close-circle-filled" 
                    class="status-icon unavailable"
                  />
                </template>
                {{ checking ? '测试中...' : '测试连接' }}
              </t-button>
              <span v-if="remoteChecked" :class="['test-message', remoteAvailable ? 'success' : 'error']">
                {{ remoteMessage }}
              </span>
            </div>
          </t-form-item>
        </template>

        <!-- Embedding 专用：维度 -->
        <t-form-item v-if="modelType === 'embedding'" label="向量维度" name="dimension">
          <t-input-number 
            v-model="formData.dimension" 
            :min="128"
            :max="4096"
            placeholder="如：1536"
          />
        </t-form-item>

        <!-- 设为默认 -->
        <t-form-item label=" " name="isDefault">
          <t-checkbox v-model="formData.isDefault">设为默认模型</t-checkbox>
        </t-form-item>
      </t-form>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { checkOllamaModels, checkRemoteModel, testEmbeddingModel, checkRerankModel, listOllamaModels, downloadOllamaModel, getDownloadProgress, type OllamaModelInfo } from '@/api/initialization'

interface ModelFormData {
  id: string
  name: string
  source: 'local' | 'remote'
  modelName: string
  baseUrl?: string
  apiKey?: string
  dimension?: number
  interfaceType?: 'ollama' | 'openai'
  isDefault: boolean
}

interface Props {
  visible: boolean
  modelType: 'chat' | 'embedding' | 'rerank' | 'vllm'
  modelData?: ModelFormData | null
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [data: ModelFormData]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const isEdit = computed(() => !!props.modelData)

const formRef = ref()
const saving = ref(false)
const modelChecked = ref(false)
const modelAvailable = ref(false)
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')

// Ollama 模型状态
const ollamaModelList = ref<OllamaModelInfo[]>([])
const loadingOllamaModels = ref(false)
const searchKeyword = ref('')
const downloading = ref(false)
const downloadProgress = ref(0)
const currentDownloadModel = ref('')
let downloadInterval: any = null

const formData = ref<ModelFormData>({
  id: '',
  name: '',
  source: 'local',
  modelName: '',
  baseUrl: '',
  apiKey: '',
  dimension: 1536,
  interfaceType: 'ollama',
  isDefault: false
})

const rules = {
  name: [{ required: true, message: '请输入模型名称' }],
  modelName: [{ required: true, message: '请输入模型标识' }],
  baseUrl: [
    { 
      required: true, 
      message: '请输入 Base URL',
      trigger: 'blur'
    }
  ]
}

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    if (props.modelData) {
      formData.value = { ...props.modelData }
    } else {
      resetForm()
    }
  }
})

// 重置表单
const resetForm = () => {
  formData.value = {
    id: generateId(),
    name: '',
    source: 'local',
    modelName: '',
    baseUrl: '',
    apiKey: '',
    dimension: props.modelType === 'embedding' ? 1536 : undefined,
    interfaceType: props.modelType === 'vllm' ? 'ollama' : undefined,
    isDefault: false
  }
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
}

// 监听来源变化，重置校验状态（已合并到下面的 watch）

// 生成唯一ID
const generateId = () => {
  return `model_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// 过滤后的模型列表
const filteredOllamaModels = computed(() => {
  if (!searchKeyword.value) return ollamaModelList.value
  return ollamaModelList.value.filter(model => 
    model.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// 是否显示"下载模型"选项
const showDownloadOption = computed(() => {
  if (!searchKeyword.value.trim()) return false
  // 检查搜索词是否已存在于模型列表中
  const exists = ollamaModelList.value.some(model => 
    model.name.toLowerCase() === searchKeyword.value.toLowerCase()
  )
  return !exists
})

// 自定义过滤逻辑（捕获搜索关键词）
const handleModelFilter = (filterWords: string) => {
  searchKeyword.value = filterWords
  return true // 让 TDesign 使用我们的 filteredOllamaModels
}

// 加载 Ollama 模型列表
const loadOllamaModels = async () => {
  if (formData.value.source !== 'local') return
  
  loadingOllamaModels.value = true
  try {
    const models = await listOllamaModels()
    ollamaModelList.value = models
  } catch (error) {
    console.error('加载 Ollama 模型列表失败:', error)
    MessagePlugin.error('加载模型列表失败')
  } finally {
    loadingOllamaModels.value = false
  }
}

// 刷新模型列表
const refreshOllamaModels = async () => {
  ollamaModelList.value = [] // 清空以强制重新加载
  await loadOllamaModels()
  MessagePlugin.success('列表已刷新')
}

// 监听下拉框可见性变化
const handleDropdownVisibleChange = (visible: boolean) => {
  if (!visible) {
    searchKeyword.value = ''
  }
}

// 格式化模型大小
const formatModelSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return ''
  const gb = bytes / (1024 * 1024 * 1024)
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

// 检查模型状态（Ollama本地模型）
const checkModelStatus = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local') {
    return
  }
  
  try {
    // 调用真实 Ollama API 检查模型是否存在
    const result = await checkOllamaModels([formData.value.modelName])
    modelChecked.value = true
    modelAvailable.value = result.models[formData.value.modelName] || false
  } catch (error) {
    console.error('检查模型状态失败:', error)
    modelChecked.value = false
    modelAvailable.value = false
  }
}

// 检查 Remote API 连接（根据模型类型调用不同的接口）
const checkRemoteAPI = async () => {
  if (!formData.value.modelName || !formData.value.baseUrl) {
    MessagePlugin.warning('请先填写模型标识和 Base URL')
    return
  }
  
  checking.value = true
  remoteChecked.value = false
  remoteMessage.value = ''
  
  try {
    let result: any
    
    // 根据模型类型调用不同的校验接口
    switch (props.modelType) {
      case 'chat':
        // 对话模型（KnowledgeQA）
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      case 'embedding':
        // Embedding 模型
        result = await testEmbeddingModel({
          source: 'remote',
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || '',
          dimension: formData.value.dimension
        })
        // 如果测试成功且返回了维度，自动填充
        if (result.available && result.dimension) {
          formData.value.dimension = result.dimension
          MessagePlugin.info(`检测到向量维度：${result.dimension}`)
        }
        break
        
      case 'rerank':
        // Rerank 模型
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      case 'vllm':
        // VLLM 模型（多模态）
        // VLLM 使用 checkRemoteModel 进行基础连接测试
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl,
          apiKey: formData.value.apiKey || ''
        })
        break
        
      default:
        MessagePlugin.error('不支持的模型类型')
        return
    }
    
    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    remoteMessage.value = result.message || (result.available ? '连接成功' : '连接失败')
    
    if (result.available) {
      MessagePlugin.success(remoteMessage.value)
    } else {
      MessagePlugin.error(remoteMessage.value)
    }
  } catch (error: any) {
    console.error('Remote API 校验失败:', error)
    remoteChecked.value = true
    remoteAvailable.value = false
    remoteMessage.value = error.message || '连接失败，请检查配置'
    MessagePlugin.error(remoteMessage.value)
  } finally {
    checking.value = false
  }
}

// 确认保存
const handleConfirm = async () => {
  try {
    await formRef.value?.validate()
    saving.value = true
    
    // 如果是新增且没有 id，生成一个
    if (!formData.value.id) {
      formData.value.id = generateId()
    }
    
    emit('confirm', { ...formData.value })
    dialogVisible.value = false
    MessagePlugin.success(isEdit.value ? '模型已更新' : '模型已添加')
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    saving.value = false
  }
}

// 监听模型选择变化（处理下载逻辑）
watch(() => formData.value.modelName, async (newValue) => {
  if (!newValue || !newValue.startsWith('__download__')) return
  
  // 提取模型名称
  const modelName = newValue.replace('__download__', '')
  
  // 重置选择（避免显示 __download__ 前缀）
  formData.value.modelName = ''
  
  // 开始下载
  await startDownload(modelName)
})

// 开始下载模型
const startDownload = async (modelName: string) => {
  downloading.value = true
  downloadProgress.value = 0
  currentDownloadModel.value = modelName
  
  try {
    // 启动下载
    const result = await downloadOllamaModel(modelName)
    const taskId = result.taskId
    
    MessagePlugin.success(`开始下载 ${modelName}`)
    
    // 轮询下载进度
    downloadInterval = setInterval(async () => {
      try {
        const progress = await getDownloadProgress(taskId)
        downloadProgress.value = progress.progress
        
        if (progress.status === 'completed') {
          // 下载完成
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          
          MessagePlugin.success(`${modelName} 下载完成`)
          
          // 刷新模型列表
          await loadOllamaModels()
          
          // 自动选中新下载的模型
          formData.value.modelName = modelName
          
          // 重置状态
          downloadProgress.value = 0
          currentDownloadModel.value = ''
          
        } else if (progress.status === 'failed') {
          // 下载失败
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          MessagePlugin.error(progress.message || `${modelName} 下载失败`)
          downloadProgress.value = 0
          currentDownloadModel.value = ''
        }
      } catch (error) {
        console.error('获取下载进度失败:', error)
      }
    }, 1000) // 每秒查询一次
    
  } catch (error: any) {
    downloading.value = false
    downloadProgress.value = 0
    currentDownloadModel.value = ''
    MessagePlugin.error(error.message || '启动下载失败')
  }
}

// 组件卸载时清理定时器
onUnmounted(() => {
  if (downloadInterval) {
    clearInterval(downloadInterval)
  }
})

// 监听来源变化，清理所有状态
watch(() => formData.value.source, () => {
  // 重置校验状态
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  
  // 清理下载状态
  searchKeyword.value = ''
  if (downloadInterval) {
    clearInterval(downloadInterval)
    downloadInterval = null
  }
  downloading.value = false
  downloadProgress.value = 0
  currentDownloadModel.value = ''
})

// 取消
const handleCancel = () => {
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
.model-editor-form {
  padding: 8px 0;
  max-height: 60vh;
  overflow-y: auto;

  // 自定义滚动条
  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: #f5f5f5;
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: #d0d0d0;
    border-radius: 3px;

    &:hover {
      background: #b0b0b0;
    }
  }

  :deep(.t-form-item) {
    margin-bottom: 18px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  :deep(.t-form-item__label) {
    font-size: 13px;
    font-weight: 500;
    color: #333333;
    padding-bottom: 6px;
    margin-bottom: 0;
  }

  :deep(.t-input),
  :deep(.t-select),
  :deep(.t-textarea),
  :deep(.t-input-number) {
    font-size: 13px;
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    transition: all 0.2s ease;

    &:hover {
      border-color: #07C05F;
      background: #fafafa;
    }

    &:focus,
    &.t-is-focused {
      background: #ffffff;
      border-color: #07C05F;
      box-shadow: 0 0 0 2px rgba(7, 192, 95, 0.1);
    }
  }

  :deep(.t-input__inner) {
    font-size: 13px;
  }

  :deep(.t-radio-group) {
    .t-radio {
      margin-right: 16px;
      font-size: 13px;

      &:last-child {
        margin-right: 0;
      }
    }

    .t-radio__label {
      font-size: 13px;
      color: #333333;
    }
  }

  :deep(.t-checkbox) {
    font-size: 13px;

    .t-checkbox__label {
      font-size: 13px;
      color: #333333;
    }
  }
}

.model-input-with-status {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }

  .status-icon {
    font-size: 16px;
    flex-shrink: 0;

    &.available {
      color: #07C05F;
    }

    &.unavailable {
      color: #e34d59;
    }
  }
}

.download-hint {
  margin-top: -10px;
  margin-bottom: 8px;

  :deep(.t-alert) {
    padding: 8px 12px;
    font-size: 12px;
    border-radius: 6px;
    background: #fff7e6;
    border-color: #ffa940;
  }
}

.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;

  .test-message {
    font-size: 12px;
    line-height: 1.5;
    
    &.success {
      color: #07C05F;
    }
    
    &.error {
      color: #e34d59;
    }
  }
  
  :deep(.t-button) {
    min-width: 100px;
  }
}

// 优化对话框样式
:deep(.t-dialog) {
  border-radius: 12px;
  
  .t-dialog__header {
    padding: 20px 24px 16px;
    border-bottom: 1px solid #e5e7eb;
    
    .t-dialog__header-content {
      font-size: 16px;
      font-weight: 600;
      color: #333333;
    }
  }

  .t-dialog__body {
    padding: 20px 24px;
  }

  .t-dialog__footer {
    padding: 12px 24px 20px;
    border-top: 1px solid #e5e7eb;
    
    .t-button {
      font-size: 13px;
      padding: 6px 16px;
      border-radius: 6px;
      
      &.t-button--theme-primary {
        background: #07C05F;
        border-color: #07C05F;
        
        &:hover {
          background: #05a34e;
          border-color: #05a34e;
        }
      }
      
      &.t-button--variant-outline {
        &:hover {
          border-color: #07C05F;
          color: #07C05F;
        }
      }
    }
  }
}

// Ollama 模型选择器样式
.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;
  
  .downloaded-icon {
    font-size: 14px;
    color: #07C05F;
    flex-shrink: 0;
  }
  
  .download-icon {
    font-size: 14px;
    color: #07C05F;
    flex-shrink: 0;
  }
  
  .model-name {
    flex: 1;
    font-size: 13px;
    color: #333333;
  }
  
  .model-size {
    font-size: 12px;
    color: #999999;
    margin-left: auto;
  }
  
  &.download {
    .model-name {
      color: #07C05F;
      font-weight: 500;
    }
  }
}

// 下载进度后缀样式
.download-suffix {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;
  
  .spinning {
    animation: spin 1s linear infinite;
    font-size: 14px;
    color: #07C05F;
  }
  
  .progress-text {
    font-size: 12px;
    font-weight: 500;
    color: #07C05F;
  }
}

// 下载中的选择框进度条效果
:deep(.t-select.downloading) {
  .t-input {
    position: relative;
    overflow: hidden;
    
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--progress, 0%);
      background: linear-gradient(90deg, rgba(7, 192, 95, 0.08), rgba(7, 192, 95, 0.15));
      transition: width 0.3s ease;
      z-index: 0;
      border-radius: 5px 0 0 5px;
    }
    
    .t-input__inner,
    input {
      position: relative;
      z-index: 1;
      background: transparent !important;
    }
  }
}

.refresh-btn {
  margin-top: 4px;
  font-size: 12px;
  color: #666666;
  
  &:hover {
    color: #07C05F;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>

