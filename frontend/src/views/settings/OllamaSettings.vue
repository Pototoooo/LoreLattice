<template>
  <div class="ollama-settings">
    <div class="section-header">
      <h2>Ollama 配置</h2>
      <p class="section-description">管理本地 Ollama 服务，查看和下载模型</p>
    </div>

    <div class="settings-group">
      <!-- 启用 Ollama -->
      <div class="setting-row">
        <div class="setting-info">
          <label>启用 Ollama</label>
          <p class="desc">启用后可以使用本地 Ollama 模型</p>
        </div>
        <div class="setting-control">
          <t-switch 
            v-model="localEnabled" 
            @change="handleEnableToggle"
            size="large"
          />
        </div>
      </div>

      <!-- Ollama 服务地址 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>Ollama 服务地址</label>
          <p class="desc">本地 Ollama 服务的 API 地址</p>
        </div>
        <div class="setting-control">
          <div class="url-control-group">
            <t-input 
              v-model="localBaseUrl" 
              placeholder="http://localhost:11434"
              :disabled="!localEnabled"
              style="flex: 1;"
            />
            <div v-if="localEnabled" class="status-indicator">
              <t-icon 
                v-if="testing"
                name="loading" 
                class="status-icon spinning"
              />
              <t-icon 
                v-else-if="connectionStatus === true"
                name="check-circle-filled" 
                class="status-icon success"
                title="连接成功"
              />
              <t-icon 
                v-else-if="connectionStatus === false"
                name="close-circle-filled" 
                class="status-icon error"
                title="连接失败"
              />
            </div>
          </div>
          <t-alert 
            v-if="connectionStatus === false && localEnabled"
            theme="error"
            message="连接失败，请检查 Ollama 是否运行"
            style="margin-top: 8px;"
          />
        </div>
      </div>

    </div>

    <!-- 已下载的模型 -->
    <div v-if="localEnabled && connectionStatus" class="model-category-section">
      <div class="category-header">
        <div class="header-info">
          <h3>已下载的模型</h3>
          <p>已安装在 Ollama 中的模型列表</p>
        </div>
        <t-button 
          size="small" 
          variant="text"
          :loading="loadingModels"
          @click="refreshModels"
        >
          <t-icon name="refresh" />刷新
        </t-button>
      </div>
      
      <div v-if="loadingModels" class="loading-state">
        <t-loading size="small" />
        <span>加载中...</span>
      </div>
      <div v-else-if="downloadedModels.length > 0" class="model-list-container">
        <div v-for="model in downloadedModels" :key="model.name" class="model-card">
          <div class="model-info">
            <div class="model-name">{{ model.name }}</div>
            <div class="model-meta">
              <span class="model-size">{{ formatSize(model.size) }}</span>
              <span class="model-modified">{{ formatDate(model.modified_at) }}</span>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">暂无已下载的模型</p>
      </div>
    </div>

    <!-- 下载新模型 -->
    <div v-if="localEnabled && connectionStatus" class="model-category-section">
      <div class="category-header">
        <div class="header-info">
          <h3>下载新模型</h3>
          <p>输入模型名称下载，或点击推荐模型快速下载</p>
        </div>
      </div>
      
      <div class="download-content">
        <div class="input-group">
          <t-input 
            v-model="downloadModelName" 
            placeholder="如：qwen2.5:0.5b"
            style="flex: 1;"
          />
          <t-button 
            theme="primary"
            size="small"
            :loading="downloading"
            :disabled="!downloadModelName.trim()"
            @click="downloadModel"
          >
            下载
          </t-button>
        </div>
        
        <div v-if="downloadProgress > 0" class="download-progress">
          <div class="progress-info">
            <span>正在下载: {{ downloadModelName }}</span>
            <span>{{ downloadProgress.toFixed(2) }}%</span>
          </div>
          <t-progress :percentage="downloadProgress" size="small" />
        </div>

        <div class="recommended-models">
          <div class="recommended-label">推荐模型：</div>
          <div class="model-tags">
            <t-tag 
              v-for="model in popularModels" 
              :key="model"
              theme="default"
              variant="outline"
              class="model-tag"
              @click="quickDownload(model)"
            >
              {{ model }}
            </t-tag>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { MessagePlugin } from 'tdesign-vue-next'
import { checkOllamaStatus, listOllamaModels, downloadOllamaModel, getDownloadProgress, type OllamaModelInfo } from '@/api/initialization'

const settingsStore = useSettingsStore()

const localEnabled = ref(settingsStore.settings.ollamaConfig?.enabled ?? true)
const localBaseUrl = ref(settingsStore.settings.ollamaConfig?.baseUrl ?? '')

const testing = ref(false)
const connectionStatus = ref<boolean | null>(null)
const loadingModels = ref(false)
const downloadedModels = ref<OllamaModelInfo[]>([])
const downloading = ref(false)
const downloadModelName = ref('')
const downloadProgress = ref(0)

// 推荐的流行模型
const popularModels = [
  'qwen2.5:0.5b',
  'qwen2.5:1.5b',
  'llama3.2:1b',
  'llama3.2:3b',
  'gemma2:2b',
  'phi3:mini'
]

// 处理启用/禁用
const handleEnableToggle = (value: boolean) => {
  settingsStore.updateOllamaConfig({ enabled: value })
  MessagePlugin.success(value ? 'Ollama 已启用' : 'Ollama 已禁用')
  
  if (value) {
    testConnection()
  } else {
    connectionStatus.value = null
  }
}

// 测试连接
const testConnection = async () => {
  if (!localEnabled.value) return
  
  testing.value = true
  connectionStatus.value = null
  
  try {
    // 保存配置
    settingsStore.updateOllamaConfig({ baseUrl: localBaseUrl.value })
    
    // 调用真实 Ollama API 测试连接
    const result = await checkOllamaStatus()
    
    // 如果接口返回了 baseUrl 且与当前输入框的值不同，更新为接口返回的值
    if (result.baseUrl && result.baseUrl !== localBaseUrl.value) {
      localBaseUrl.value = result.baseUrl
      settingsStore.updateOllamaConfig({ baseUrl: result.baseUrl })
    }
    
    connectionStatus.value = result.available
    
    if (connectionStatus.value) {
      MessagePlugin.success('连接成功')
      refreshModels()
    } else {
      MessagePlugin.error(result.error || '连接失败，请检查 Ollama 是否运行')
    }
  } catch (error: any) {
    connectionStatus.value = false
    MessagePlugin.error(error.message || '连接失败')
  } finally {
    testing.value = false
  }
}

// 刷新模型列表
const refreshModels = async () => {
  loadingModels.value = true
  
  try {
    // 调用真实 Ollama API 获取模型列表（现在返回完整的模型信息）
    const models = await listOllamaModels()
    downloadedModels.value = models
  } catch (error: any) {
    console.error('获取模型列表失败:', error)
    MessagePlugin.error(error.message || '获取模型列表失败')
  } finally {
    loadingModels.value = false
  }
}

// 格式化文件大小
const formatSize = (bytes: number): string => {
  if (!bytes || bytes === 0 || isNaN(bytes)) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

// 格式化日期
const formatDate = (dateStr: string): string => {
  if (!dateStr) return '未知'
  
  const date = new Date(dateStr)
  // 检查日期是否有效
  if (isNaN(date.getTime())) return '未知'
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days} 天前`
  if (days < 0) return date.toLocaleDateString('zh-CN')
  return date.toLocaleDateString('zh-CN')
}

// 下载模型
const downloadModel = async () => {
  if (!downloadModelName.value.trim()) return
  
  downloading.value = true
  downloadProgress.value = 0
  
  try {
    // 调用真实 Ollama API 下载模型
    const result = await downloadOllamaModel(downloadModelName.value)
    
    if (result.status === 'failed') {
      MessagePlugin.error('下载失败，请稍后重试')
      downloading.value = false
      downloadProgress.value = 0
      return
    }
    
    MessagePlugin.success(`已开始下载模型 ${downloadModelName.value}`)
    
    // 查询下载进度
    const taskId = result.taskId
    const progressInterval = setInterval(async () => {
      try {
        const task = await getDownloadProgress(taskId)
        downloadProgress.value = task.progress
        
        if (task.status === 'completed') {
          clearInterval(progressInterval)
          MessagePlugin.success(`模型 ${downloadModelName.value} 下载完成`)
          downloadModelName.value = ''
          downloadProgress.value = 0
          downloading.value = false
          refreshModels()
        } else if (task.status === 'failed') {
          clearInterval(progressInterval)
          MessagePlugin.error(task.message || '下载失败')
          downloading.value = false
          downloadProgress.value = 0
        }
      } catch (error) {
        clearInterval(progressInterval)
        MessagePlugin.error('查询下载进度失败')
        downloading.value = false
        downloadProgress.value = 0
      }
    }, 1000)
  } catch (error: any) {
    console.error('下载失败:', error)
    MessagePlugin.error(error.message || '下载失败')
    downloading.value = false
    downloadProgress.value = 0
  }
}

// 快速下载推荐模型
const quickDownload = (modelName: string) => {
  downloadModelName.value = modelName
  downloadModel()
}

// 初始化 Ollama 服务地址
const initOllamaBaseUrl = async () => {
  try {
    const result = await checkOllamaStatus()
    // 如果接口返回了 baseUrl，优先使用接口返回的值
    if (result.baseUrl) {
      localBaseUrl.value = result.baseUrl
      // 如果 store 中没有保存过，也保存到 store 中
      if (!settingsStore.settings.ollamaConfig?.baseUrl) {
        settingsStore.updateOllamaConfig({ baseUrl: result.baseUrl })
      }
    } else if (!localBaseUrl.value) {
      // 如果接口没返回且 store 中也没有，使用默认值
      localBaseUrl.value = 'http://localhost:11434'
    }
    
    // 如果启用了，直接使用初始化时获取的状态，避免重复调用
    if (localEnabled.value) {
      connectionStatus.value = result.available
      if (result.available) {
        refreshModels()
      }
    }
    
    return result
  } catch (error) {
    console.error('初始化 Ollama 地址失败:', error)
    // 如果获取失败，使用默认值或 store 中的值
    if (!localBaseUrl.value) {
      localBaseUrl.value = 'http://localhost:11434'
    }
    return null
  }
}

// 组件挂载时自动检查连接
onMounted(async () => {
  // 初始化服务地址，如果启用则直接使用返回的状态，避免重复调用
  await initOllamaBaseUrl()
})
</script>

<style lang="less" scoped>
.ollama-settings {
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
  min-width: 420px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.url-control-group {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 4px;

    .status-icon {
      font-size: 18px;

      &.success {
        color: var(--td-success-color);
      }

      &.error {
        color: var(--td-error-color);
      }

      &.spinning {
        animation: spin 1s linear infinite;
      }
    }
  }
}

.model-category-section {
  margin-bottom: 24px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  padding: 24px;
  background: var(--td-bg-color-container);

  &:last-child {
    margin-bottom: 0;
  }
}

.category-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;

  .header-info {
    flex: 1;

    h3 {
      font-size: 16px;
      font-weight: 600;
      color: var(--td-text-color-primary);
      margin: 0 0 4px 0;
    }

    p {
      font-size: 14px;
      color: var(--td-text-color-secondary);
      margin: 0;
      line-height: 1.5;
    }
  }
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 60px;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.model-list-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.model-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  transition: all 0.2s;

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container);
  }
}

.model-info {
  flex: 1;
  min-width: 0;

  .model-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 4px;
    font-family: monospace;
  }

  .model-meta {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

.download-content {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .input-group {
    display: flex;
    gap: 8px;
  }

  .download-progress {
    padding: 12px;
    background: var(--td-bg-color-component);
    border-radius: 6px;

    .progress-info {
      display: flex;
      justify-content: space-between;
      margin-bottom: 8px;
      font-size: 13px;
      color: var(--td-text-color-primary);
    }
  }

  .recommended-models {
    .recommended-label {
      font-size: 13px;
      color: var(--td-text-color-secondary);
      margin: 0 0 10px 0;
      font-weight: 500;
    }

    .model-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;

      .model-tag {
        cursor: pointer;
        transition: all 0.2s;
        font-size: 12px;

        &:hover {
          background: var(--td-brand-color);
          color: var(--td-text-color-anti);
          border-color: var(--td-brand-color);
        }
      }
    }
  }
}

.empty-state {
  padding: 80px 0;
  text-align: center;

  .empty-text {
    font-size: 14px;
    color: var(--td-text-color-placeholder);
    margin: 0;
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
