<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="dialogVisible" class="model-editor-overlay" @click.self="handleCancel">
        <div class="model-editor-modal">
          <!-- 关闭按钮 -->
          <button class="close-btn" @click="handleCancel" aria-label="关闭">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>

          <!-- 标题区域 -->
          <div class="modal-header">
            <h2 class="modal-title">{{ isEdit ? '编辑模型' : '添加模型' }}</h2>
            <p class="modal-desc">{{ getModalDescription() }}</p>
          </div>

          <!-- 表单内容区域 -->
          <div class="modal-body">
            <t-form ref="formRef" :data="formData" :rules="rules" layout="vertical">
        <!-- 模型来源 -->
        <div class="form-item">
          <label class="form-label required">模型来源</label>
          <t-radio-group v-model="formData.source">
            <t-radio value="local">Ollama (本地)</t-radio>
            <t-radio value="remote">Remote API (远程)</t-radio>
          </t-radio-group>
        </div>

        <!-- Ollama 本地模型选择器 -->
        <div v-if="formData.source === 'local'" class="form-item">
          <label class="form-label required">模型名称</label>
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
        </div>

        <!-- Remote API 和 VLLM 保持原有的 input -->
        <div v-else class="form-item">
          <label class="form-label required">模型名称</label>
          <t-input 
            v-model="formData.modelName" 
            :placeholder="getModelNamePlaceholder()"
          />
        </div>

        <!-- Remote API 配置 -->
        <template v-if="formData.source === 'remote'">
          <div class="form-item">
            <label class="form-label required">Base URL</label>
            <t-input 
              v-model="formData.baseUrl" 
              :placeholder="modelType === 'vllm' ? '如：http://localhost:11434/v1' : '如：https://api.openai.com/v1'"
            />
          </div>

          <div class="form-item">
            <label class="form-label">API Key (可选)</label>
            <t-input 
              v-model="formData.apiKey" 
              type="password"
              placeholder="输入 API Key"
            />
          </div>

          <!-- Remote API 校验 -->
          <div class="form-item">
            <label class="form-label">连接测试</label>
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
          </div>
        </template>

        <!-- Embedding 专用：维度 -->
        <div v-if="modelType === 'embedding'" class="form-item">
          <label class="form-label">向量维度</label>
          <t-input-number 
            v-model="formData.dimension" 
            :min="128"
            :max="4096"
            placeholder="如：1536"
          />
        </div>

        <!-- 设为默认 -->
        <div class="form-item">
          <t-checkbox v-model="formData.isDefault">设为默认模型</t-checkbox>
        </div>
      </t-form>
          </div>

          <!-- 底部按钮区域 -->
          <div class="modal-footer">
            <t-button theme="default" variant="outline" @click="handleCancel">
              取消
            </t-button>
            <t-button theme="primary" @click="handleConfirm" :loading="saving">
              保存
            </t-button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
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
  modelName: [
    { required: true, message: '请输入模型名称' },
    { 
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: '模型名称不能为空' }
        }
        if (val.trim().length > 100) {
          return { result: false, message: '模型名称不能超过100个字符' }
        }
        return { result: true }
      },
      trigger: 'blur'
    }
  ],
  baseUrl: [
    { 
      required: true, 
      message: '请输入 Base URL',
      trigger: 'blur'
    },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: 'Base URL 不能为空' }
        }
        // 简单的 URL 格式校验
        try {
          new URL(val.trim())
          return { result: true }
        } catch {
          return { result: false, message: 'Base URL 格式不正确，请输入有效的 URL' }
        }
      },
      trigger: 'blur'
    }
  ]
}

// 获取弹窗描述文字
const getModalDescription = () => {
  const typeDesc = {
    chat: '配置用于对话的大语言模型',
    embedding: '配置用于文本向量化的嵌入模型',
    rerank: '配置用于结果重排序的模型',
    vllm: '配置用于视觉理解和多模态的视觉语言模型'
  }
  return typeDesc[props.modelType] || '配置模型信息'
}

// 获取模型名称占位符
const getModelNamePlaceholder = () => {
  if (props.modelType === 'vllm') {
    return formData.value.source === 'local' ? '如：llava:latest' : '如：gpt-4-vision-preview'
  }
  return formData.value.source === 'local' ? '如：llama2:latest' : '如：gpt-4, claude-3-opus'
}

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    // 锁定背景滚动
    document.body.style.overflow = 'hidden'
    
    if (props.modelData) {
      formData.value = { ...props.modelData }
    } else {
      resetForm()
    }
  } else {
    // 恢复背景滚动
    document.body.style.overflow = ''
  }
})

// 重置表单
const resetForm = () => {
  formData.value = {
    id: generateId(),
    name: '', // 保留字段但不使用，保存时用 modelName
    source: 'local',
    modelName: '',
    baseUrl: '',
    apiKey: '',
    dimension: props.modelType === 'embedding' ? 1536 : undefined,
    interfaceType: undefined,
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
  // 只在选择 local 来源时加载
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
    // 手动校验必填字段
    if (!formData.value.modelName || !formData.value.modelName.trim()) {
      MessagePlugin.warning('请输入模型名称')
      return
    }
    
    if (formData.value.modelName.trim().length > 100) {
      MessagePlugin.warning('模型名称不能超过100个字符')
      return
    }
    
    // 如果是 remote 类型，必须填写 baseUrl
    if (formData.value.source === 'remote') {
      if (!formData.value.baseUrl || !formData.value.baseUrl.trim()) {
        MessagePlugin.warning('Remote API 类型必须填写 Base URL')
        return
      }
      
      // 校验 Base URL 格式
      try {
        new URL(formData.value.baseUrl.trim())
      } catch {
        MessagePlugin.warning('Base URL 格式不正确，请输入有效的 URL')
        return
      }
    }
    
    // 执行表单验证
    await formRef.value?.validate()
    saving.value = true
    
    // 如果是新增且没有 id，生成一个
    if (!formData.value.id) {
      formData.value.id = generateId()
    }
    
    emit('confirm', { ...formData.value })
    dialogVisible.value = false
    // 移除此处的成功提示，由父组件统一处理
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
// 遮罩层
.model-editor-overlay {
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
  overflow: hidden; // 防止背景滚动
}

// 弹窗主体
.model-editor-modal {
  position: relative;
  width: 90vw;
  max-width: 600px;
  max-height: 85vh;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

// 关闭按钮
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

// 标题区域
.modal-header {
  padding: 24px 32px 20px;
  border-bottom: 1px solid #e5e5e5;
  flex-shrink: 0;
}

.modal-title {
  margin: 0 0 8px 0;
  font-family: "PingFang SC";
  font-size: 18px;
  font-weight: 600;
  color: #000000e6;
}

.modal-desc {
  margin: 0;
  font-family: "PingFang SC";
  font-size: 14px;
  color: #00000066;
  line-height: 22px;
}

// 内容区域
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px 32px;

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

  :deep(.t-form) {
    .t-form-item {
      display: none; // 隐藏 t-form-item，使用自定义的 form-item
    }
  }
}

// 表单项样式
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

// 输入框样式
:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: 14px;
  
  .t-input__inner,
  input {
    font-size: 14px;
  }
}

// 单选按钮组
:deep(.t-radio-group) {
  display: flex;
  gap: 16px;
  
  .t-radio {
    margin-right: 0;
    font-size: 14px;
  }

  .t-radio__label {
    font-size: 14px;
    color: #000000e6;
  }
}

// 复选框
:deep(.t-checkbox) {
  font-size: 14px;

  .t-checkbox__label {
    font-size: 14px;
    color: #000000e6;
  }
}

// 底部按钮区域
.modal-footer {
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

  .model-editor-modal {
    transform: scale(0.95);
  }
}

// API 测试区域
.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;

  .test-message {
    font-size: 13px;
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

