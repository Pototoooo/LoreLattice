<template>
  <div class="agent-settings">
    <div class="section-header">
      <h2>Agent 配置</h2>
      <p class="section-description">配置 AI Agent 的默认行为和参数，这些设置将应用于所有启用 Agent 模式的对话</p>
      
      <!-- Agent 状态显示 -->
      <div class="agent-status-row">
        <div class="status-label">
          <label>Agent 状态</label>
        </div>
        <div class="status-control">
          <div class="status-badge" :class="{ ready: isAgentReady }">
            <t-icon 
              v-if="isAgentReady" 
              name="check-circle-filled" 
              class="status-icon"
            />
            <t-icon 
              v-else 
              name="error-circle-filled" 
              class="status-icon"
            />
            <span class="status-text">
              {{ isAgentReady ? '可用' : '未就绪' }}
            </span>
          </div>
          <span v-if="!isAgentReady" class="status-hint">
            {{ agentStatusMessage }}
          </span>
          <p v-if="!isAgentReady" class="status-tip">
            <t-icon name="info-circle" class="tip-icon" />
            配置完成后，Agent 状态将自动变为"可用"，此时可在对话界面开启 Agent 模式
          </p>
        </div>
      </div>
      </div>

    <div class="settings-group">

      <!-- 最大迭代次数 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>最大迭代次数</label>
          <p class="desc">Agent 执行任务时的最大推理步骤数</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
          <t-slider 
            v-model="localMaxIterations" 
            :min="1" 
            :max="30" 
            :step="1"
            :marks="{ 1: '1', 5: '5', 10: '10', 15: '15', 20: '20', 25: '25', 30: '30' }"
            @change="handleMaxIterationsChangeDebounced"
              style="width: 200px;"
          />
            <span class="value-display">{{ localMaxIterations }}</span>
          </div>
        </div>
      </div>

      <!-- 思考模型 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>思考模型</label>
          <p class="desc">用于 Agent 推理和规划的 LLM 模型</p>
        </div>
        <div class="setting-control">
        <t-select
          v-model="localThinkingModelId"
          :loading="loadingModels"
          filterable
          placeholder="搜索模型..."
          @change="handleThinkingModelChange"
          @focus="loadAllModels"
          style="width: 280px;"
        >
          <!-- 已有的对话模型 -->
          <t-option
            v-for="model in chatModels"
            :key="model.id"
            :value="model.id"
            :label="model.name"
          >
            <div class="model-option">
              <t-icon name="check-circle-filled" class="model-icon" />
              <span class="model-name">{{ model.name }}</span>
              <t-tag v-if="model.is_default" size="small" theme="success">默认</t-tag>
            </div>
          </t-option>
          
          <!-- 添加模型选项 -->
          <t-option value="__add_model__" class="add-model-option">
            <div class="model-option add">
              <t-icon name="add" class="add-icon" />
              <span class="model-name">添加新的对话模型</span>
            </div>
          </t-option>
        </t-select>
        </div>
      </div>

      <!-- Rerank 模型 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>Rerank 模型</label>
          <p class="desc">搜索结果重排序，统一不同来源的相关度分数</p>
        </div>
        <div class="setting-control">
        <t-select
          v-model="localRerankModelId"
          :loading="loadingModels"
          filterable
          placeholder="搜索模型..."
          @change="handleRerankModelChange"
          @focus="loadAllModels"
          style="width: 280px;"
        >
          <!-- 已有的 Rerank 模型 -->
          <t-option
            v-for="model in rerankModels"
            :key="model.id"
            :value="model.id"
            :label="model.name"
          >
            <div class="model-option">
              <t-icon name="check-circle-filled" class="model-icon" />
              <span class="model-name">{{ model.name }}</span>
              <t-tag v-if="model.is_default" size="small" theme="success">默认</t-tag>
            </div>
          </t-option>
          
          <!-- 添加模型选项 -->
          <t-option value="__add_model__" class="add-model-option">
            <div class="model-option add">
              <t-icon name="add" class="add-icon" />
              <span class="model-name">添加新的 Rerank 模型</span>
            </div>
          </t-option>
        </t-select>
        </div>
      </div>

      <!-- 温度参数 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>温度参数</label>
          <p class="desc">控制模型输出的随机性，0 最确定，1 最随机</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
          <t-slider 
            v-model="localTemperature" 
            :min="0" 
            :max="1" 
            :step="0.1"
            :marks="{ 0: '0', 0.5: '0.5', 1: '1' }"
            @change="handleTemperatureChange"
              style="width: 200px;"
          />
            <span class="value-display">{{ localTemperature.toFixed(1) }}</span>
          </div>
        </div>
      </div>

      <!-- 允许的工具 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>允许的工具</label>
          <p class="desc">选择 Agent 可以使用的工具，至少选择一个</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localAllowedTools"
            multiple
            placeholder="请选择工具..."
            @change="handleAllowedToolsChange"
            style="width: 400px;"
          >
            <t-option
              v-for="tool in availableTools"
              :key="tool.name"
              :value="tool.name"
              :label="tool.label"
              :title="tool.description"
            >
              {{ tool.label }}
            </t-option>
          </t-select>
        </div>
      </div>

      <!-- 系统 Prompt -->
      <div class="setting-row vertical">
        <div class="setting-info">
          <label>系统 Prompt</label>
          <p class="desc">配置 Agent 的系统提示词，支持占位符模板。占位符会在运行时自动替换为实际内容。</p>
          <div class="placeholder-hint">
            <p class="hint-title">可用占位符：</p>
            <ul class="placeholder-list">
              <li v-for="placeholder in availablePlaceholders" :key="placeholder.name">
                <code v-html="`{{${placeholder.name}}}`"></code> - {{ placeholder.label }}（{{ placeholder.description }}）
              </li>
            </ul>
            <p class="hint-tip">提示：输入 <code>&#123;&#123;</code> 时会自动显示可用占位符</p>
          </div>
        </div>
        <div class="setting-control full-width" style="position: relative;">
          <div class="prompt-header">
            <t-button
              theme="default"
              variant="outline"
              size="small"
              @click="handleResetToDefault"
              :loading="isResettingPrompt"
            >
              恢复默认
            </t-button>
          </div>
          <div class="prompt-textarea-wrapper">
            <t-textarea
              ref="promptTextareaRef"
              v-model="localSystemPrompt"
              :autosize="{ minRows: 15, maxRows: 30 }"
              placeholder="请输入系统 Prompt，或留空使用默认 Prompt..."
              @blur="handleSystemPromptChange"
              @input="handlePromptInput"
              @keydown="handlePromptKeydown"
              style="width: 100%; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 13px;"
            />
          </div>
          <!-- 占位符提示下拉框 -->
          <teleport to="body">
            <div v-if="showPlaceholderPopup && filteredPlaceholders.length > 0" class="placeholder-popup-wrapper" :style="popupStyle">
              <div class="placeholder-popup">
              <div
                v-for="(placeholder, index) in filteredPlaceholders"
                :key="placeholder.name"
                class="placeholder-item"
                :class="{ active: selectedPlaceholderIndex === index }"
                @mousedown.prevent="insertPlaceholder(placeholder.name)"
                @mouseenter="selectedPlaceholderIndex = index"
              >
                  <div class="placeholder-name">
                    <code v-html="`{{${placeholder.name}}}`"></code>
                  </div>
                  <div class="placeholder-desc">{{ placeholder.description }}</div>
                </div>
              </div>
            </div>
          </teleport>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { listModels, type ModelConfig } from '@/api/model'
import { getAgentConfig, updateAgentConfig, type AgentConfig, type ToolDefinition, type PlaceholderDefinition } from '@/api/system'

const settingsStore = useSettingsStore()
const router = useRouter()

// 本地状态
const localMaxIterations = ref(5)
const localTemperature = ref(0.7)
const localThinkingModelId = ref('')
const localRerankModelId = ref('')
const localAllowedTools = ref<string[]>([])
const localSystemPrompt = ref('')

// 计算 Agent 是否就绪
const isAgentReady = computed(() => {
  // 必须有思考模型、Rerank 模型 且 至少选择一个工具
  return localThinkingModelId.value !== '' && 
         localRerankModelId.value !== '' && 
         localAllowedTools.value.length > 0
})

// Agent 状态提示消息
const agentStatusMessage = computed(() => {
  const missing: string[] = []
  
  if (!localThinkingModelId.value) {
    missing.push('思考模型')
  }
  if (!localRerankModelId.value) {
    missing.push('Rerank 模型')
  }
  if (localAllowedTools.value.length === 0) {
    missing.push('允许的工具')
  }
  
  if (missing.length === 0) {
    return ''
  }
  
  return `请配置${missing.join('、')}`
})

// 模型列表状态
const chatModels = ref<ModelConfig[]>([])
const rerankModels = ref<ModelConfig[]>([])
const loadingModels = ref(false)

// 可用工具列表
const availableTools = ref<ToolDefinition[]>([])
// 可用占位符列表
const availablePlaceholders = ref<PlaceholderDefinition[]>([])

// 配置加载状态
const loadingConfig = ref(false)
const configLoaded = ref(false) // 防止重复加载
const isInitializing = ref(true) // 标记是否正在初始化，防止初始化时触发保存

// 保存的 Prompt 值，用于比较是否变化
let savedSystemPrompt = ''

// 恢复默认 Prompt 的加载状态
const isResettingPrompt = ref(false)

// 占位符提示相关状态
const promptTextareaRef = ref<any>(null)
const showPlaceholderPopup = ref(false)
const selectedPlaceholderIndex = ref(0)
let placeholderPopupTimer: any = null
const placeholderPrefix = ref('') // 当前输入的前缀，用于过滤
const popupStyle = ref({ top: '0px', left: '0px' }) // 提示框位置

// 设置 textarea 原生事件监听器
const setupTextareaEventListeners = () => {
  nextTick(() => {
    const textarea = getTextareaElement()
    if (textarea) {
      // 添加原生 keydown 事件监听（使用 capture 阶段，确保优先处理）
      textarea.addEventListener('keydown', (e: KeyboardEvent) => {
        // 如果正在显示占位符提示，优先处理占位符相关的按键
        if (showPlaceholderPopup.value && filteredPlaceholders.value.length > 0) {
          if (e.key === 'ArrowDown') {
            // 下箭头选择下一个
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            if (selectedPlaceholderIndex.value < filteredPlaceholders.value.length - 1) {
              selectedPlaceholderIndex.value++
            } else {
              selectedPlaceholderIndex.value = 0 // 循环到第一个
            }
            return
          } else if (e.key === 'ArrowUp') {
            // 上箭头选择上一个
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            if (selectedPlaceholderIndex.value > 0) {
              selectedPlaceholderIndex.value--
            } else {
              selectedPlaceholderIndex.value = filteredPlaceholders.value.length - 1 // 循环到最后一个
            }
            return
          } else if (e.key === 'Enter') {
            // Enter 键插入选中的占位符
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            const selected = filteredPlaceholders.value[selectedPlaceholderIndex.value]
            if (selected) {
              insertPlaceholder(selected.name)
            }
            return
          } else if (e.key === 'Escape') {
            // ESC 键关闭提示
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            showPlaceholderPopup.value = false
            placeholderPrefix.value = ''
            return
          }
        }
        
        // 如果按下的是 { 键
        if (e.key === '{') {
          // 清除之前的定时器
          if (placeholderPopupTimer) {
            clearTimeout(placeholderPopupTimer)
          }
          
          // 延迟检查，等待输入完成（连续输入两个 {）
          placeholderPopupTimer = setTimeout(() => {
            checkAndShowPlaceholderPopup()
          }, 150)
        }
      }, true) // 使用 capture 阶段
      
      // 添加原生 input 事件监听（作为备用）
      textarea.addEventListener('input', () => {
        if (placeholderPopupTimer) {
          clearTimeout(placeholderPopupTimer)
        }
        placeholderPopupTimer = setTimeout(() => {
          checkAndShowPlaceholderPopup()
        }, 50)
      })
    }
  })
}

// 获取 textarea 元素的辅助函数
const getTextareaElement = (): HTMLTextAreaElement | null => {
  if (promptTextareaRef.value) {
    if (promptTextareaRef.value.$el) {
      return promptTextareaRef.value.$el.querySelector('textarea')
    } else if (promptTextareaRef.value instanceof HTMLTextAreaElement) {
      return promptTextareaRef.value
    }
  }
  
  // 如果还是找不到，尝试通过 DOM 查找
  const wrapper = document.querySelector('.setting-control.full-width')
  return wrapper?.querySelector('textarea') || null
}

// 初始化加载
onMounted(async () => {
  // 防止重复加载
  if (configLoaded.value) return
  
  loadingConfig.value = true
  configLoaded.value = true
  isInitializing.value = true
  
  try {
    // 从后台加载配置
    const res = await getAgentConfig()
    const config = res.data
    
    // 更新本地状态（在初始化期间，不会触发保存）
    localMaxIterations.value = config.max_iterations
    lastSavedValue = config.max_iterations // 初始化时记录已保存的值
    localTemperature.value = config.temperature
    localThinkingModelId.value = config.thinking_model_id
    localRerankModelId.value = config.rerank_model_id
    localAllowedTools.value = config.allowed_tools || []
    localSystemPrompt.value = config.system_prompt || ''
    savedSystemPrompt = config.system_prompt || '' // 记录已保存的值
    availableTools.value = config.available_tools || []
    availablePlaceholders.value = config.available_placeholders || []
    
    // 调试信息
    console.log('加载的占位符列表:', availablePlaceholders.value)
    
    // 统一加载所有模型（只调用一次API）
    if (config.thinking_model_id || config.rerank_model_id) {
      await loadAllModels()
    }
    
    // 同步到store（只更新本地存储，不触发API保存）
    // 注意：不自动设置 isAgentEnabled，保持用户之前的选择
    // enabled 状态应该由用户手动控制，而不是根据配置自动设置
    settingsStore.updateAgentConfig({
      maxIterations: config.max_iterations,
      temperature: config.temperature,
      thinkingModelId: config.thinking_model_id,
      rerankModelId: config.rerank_model_id,
      allowedTools: config.allowed_tools || []
    })
    
    // 等待下一个 tick，确保所有响应式更新完成
    await nextTick()
    // 再等待一帧，确保所有事件监听器都已设置好
    requestAnimationFrame(() => {
      // 初始化完成，现在可以允许保存操作
      isInitializing.value = false
      
      // 设置原生事件监听器（作为备用方案）
      setupTextareaEventListeners()
    })
  } catch (error) {
    console.error('加载Agent配置失败:', error)
    MessagePlugin.error('加载Agent配置失败')
    configLoaded.value = false // 加载失败时重置标记，允许重试
    
    // 失败时从store加载
    localMaxIterations.value = settingsStore.agentConfig.maxIterations
    localTemperature.value = settingsStore.agentConfig.temperature
    localThinkingModelId.value = settingsStore.agentConfig.thinkingModelId
    localRerankModelId.value = settingsStore.agentConfig.rerankModelId
  } finally {
    loadingConfig.value = false
    isInitializing.value = false // 确保初始化完成，即使失败也要允许后续操作
  }
})

// 错误码到错误消息的映射
const getErrorMessage = (error: any): string => {
  const errorCode = error?.response?.data?.error?.code
  const errorMessage = error?.response?.data?.error?.message
  
  switch (errorCode) {
    case 2100:
      return '启用Agent模式前，请先选择思考模型'
    case 2101:
      return '至少需要选择一个允许的工具'
    case 2102:
      return '最大迭代次数必须在1-20之间'
    case 2103:
      return '温度参数必须在0-2之间'
    case 1010:
      return errorMessage || '配置验证失败'
    default:
      return errorMessage || '保存失败，请重试'
  }
}

// 防抖定时器
let maxIterationsDebounceTimer: any = null
// 上次保存的值，用于避免重复保存相同值
let lastSavedValue: number | null = null

// 处理最大迭代次数变化（防抖版本，点击和拖动都使用这个）
const handleMaxIterationsChangeDebounced = (value: number) => {
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  // 确保 value 是数字类型
  const numValue = typeof value === 'number' ? value : Number(value)
  if (isNaN(numValue)) {
    console.error('Invalid max_iterations value:', value)
    return
  }
  
  // 如果值没有变化，不保存
  if (lastSavedValue === numValue) {
    return
  }
  
  // 清除之前的定时器
  if (maxIterationsDebounceTimer) {
    clearTimeout(maxIterationsDebounceTimer)
}

  // 设置新的定时器，300ms 后保存（减少延迟，提升响应速度）
  maxIterationsDebounceTimer = setTimeout(async () => {
    // 再次检查值是否变化（可能在等待期间值又变了）
    if (lastSavedValue === numValue) {
      maxIterationsDebounceTimer = null
      return
    }
  
  try {
    const config: AgentConfig = {
        enabled: isAgentReady.value, // 自动根据配置状态设置
        max_iterations: numValue, // 确保是数字类型
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
      settingsStore.updateAgentConfig({ maxIterations: numValue })
      lastSavedValue = numValue // 记录已保存的值
    MessagePlugin.success('最大迭代次数已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
    } finally {
      maxIterationsDebounceTimer = null
  }
  }, 300)
}

// 统一加载所有模型（只调用一次API）
const loadAllModels = async () => {
  if (chatModels.value.length > 0 && rerankModels.value.length > 0) return // 已经加载过
  
  loadingModels.value = true
  try {
    const allModels = await listModels()
    // 按类型过滤，避免重复调用
    chatModels.value = allModels.filter(m => m.type === 'KnowledgeQA')
    rerankModels.value = allModels.filter(m => m.type === 'Rerank')
  } catch (error) {
    console.error('加载模型列表失败:', error)
    MessagePlugin.error('加载模型列表失败')
  } finally {
    loadingModels.value = false
  }
}

// 加载对话模型列表（已废弃，使用 loadAllModels）
const loadChatModels = async () => {
  await loadAllModels()
}

// 加载 Rerank 模型列表（已废弃，使用 loadAllModels）
const loadRerankModels = async () => {
  await loadAllModels()
}

// 处理思考模型变化
const handleThinkingModelChange = async (value: string) => {
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  // 如果选择添加新模型，跳转到模型配置页
  if (value === '__add_model__') {
    router.push('/settings?section=models')
    return
  }
  
  try {
    const config: AgentConfig = {
      enabled: isAgentReady.value, // 自动根据配置状态设置
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: value,
      rerank_model_id: localRerankModelId.value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
    // 更新 store，确保 isAgentReady 能正确计算
    settingsStore.updateAgentConfig({ thinkingModelId: value })
    MessagePlugin.success('思考模型已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// 监听模型选择，处理"添加模型"跳转
// 处理 Rerank 模型变化
const handleRerankModelChange = async (value: string) => {
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  // 如果选择添加新模型，跳转到模型配置页
  if (value === '__add_model__') {
    router.push('/settings?section=models&subsection=rerank')
    return
  }
  
  try {
    const config: AgentConfig = {
      enabled: isAgentReady.value, // 自动根据配置状态设置
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ rerankModelId: value })
    MessagePlugin.success('Rerank 模型已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
    // 回滚
    localRerankModelId.value = settingsStore.agentConfig.rerankModelId
  }
}

watch(() => localThinkingModelId.value, (newValue) => {
  if (newValue === '__add_model__') {
    // 重置选择
    localThinkingModelId.value = ''
    
    // 跳转到模型配置页面的对话模型部分
    router.push('/platform/settings')
    
    // 发送导航事件，定位到对话模型
    setTimeout(() => {
      const event = new CustomEvent('settings-nav', { 
        detail: { section: 'models', subsection: 'chat' }
      })
      window.dispatchEvent(event)
      
      // 滚动到对话模型区域
      setTimeout(() => {
        const element = document.querySelector('[data-model-type="chat"]')
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }, 200)
    }, 100)
  }
})

// 监听 Rerank 模型选择，处理"添加模型"跳转
watch(() => localRerankModelId.value, (newValue) => {
  if (newValue === '__add_model__') {
    // 重置选择
    localRerankModelId.value = ''
    
    // 跳转到模型配置页面的 Rerank 模型部分
    router.push('/platform/settings')
    
    // 发送导航事件，定位到 Rerank 模型
    setTimeout(() => {
      const event = new CustomEvent('settings-nav', { 
        detail: { section: 'models', subsection: 'rerank' }
      })
      window.dispatchEvent(event)
      
      // 滚动到 Rerank 模型区域
      setTimeout(() => {
        const element = document.querySelector('[data-model-type="rerank"]')
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }, 200)
    }, 100)
  }
})

// 处理温度参数变化
const handleTemperatureChange = async (value: number) => {
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  try {
    const config: AgentConfig = {
      enabled: isAgentReady.value, // 自动根据配置状态设置
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ temperature: value })
    MessagePlugin.success('温度参数已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// 处理允许工具变化
const handleAllowedToolsChange = async (value: string[]) => {
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  try {
    const config: AgentConfig = {
      enabled: isAgentReady.value, // 自动根据配置状态设置
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ allowedTools: value })
    MessagePlugin.success('工具配置已更新')
  } catch (error) {
    console.error('保存工具配置失败:', error)
    MessagePlugin.error(getErrorMessage(error))
    // 回滚
    localAllowedTools.value = settingsStore.agentConfig.allowedTools
  }
}

// 处理系统 Prompt 键盘事件（作为备用，主要逻辑在原生事件监听器中）
const handlePromptKeydown = (e: KeyboardEvent) => {
  // 如果正在显示占位符提示，且输入的是字母、数字或下划线，实时更新过滤
  if (showPlaceholderPopup.value && /^[a-zA-Z0-9_]$/.test(e.key)) {
    // 延迟检查，等待字符输入完成
    if (placeholderPopupTimer) {
      clearTimeout(placeholderPopupTimer)
    }
    placeholderPopupTimer = setTimeout(() => {
      checkAndShowPlaceholderPopup()
    }, 50)
  }
}

// 过滤后的占位符列表（根据前缀匹配）
const filteredPlaceholders = computed(() => {
  if (!placeholderPrefix.value) {
    return availablePlaceholders.value
  }
  
  const prefix = placeholderPrefix.value.toLowerCase()
  return availablePlaceholders.value.filter(p => 
    p.name.toLowerCase().startsWith(prefix)
  )
})

// 计算光标在 textarea 中的像素位置
const calculateCursorPosition = (textarea: HTMLTextAreaElement) => {
  const cursorPos = textarea.selectionStart
  const textBeforeCursor = localSystemPrompt.value.substring(0, cursorPos)
  
  // 获取 textarea 的样式和位置
  const style = window.getComputedStyle(textarea)
  const textareaRect = textarea.getBoundingClientRect()
  
  // 计算行数和当前行的文本
  const lines = textBeforeCursor.split('\n')
  const currentLine = lines.length - 1
  const lineText = lines[currentLine] || ''
  
  // 获取行高
  const lineHeight = parseFloat(style.lineHeight) || parseFloat(style.fontSize) * 1.2
  
  // 获取 padding
  const paddingTop = parseFloat(style.paddingTop) || 0
  const paddingLeft = parseFloat(style.paddingLeft) || 0
  
  // 使用 canvas 测量当前行的文本宽度（更准确）
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')
  let textWidth = 0
  
  if (context) {
    context.font = `${style.fontSize} ${style.fontFamily}`
    textWidth = context.measureText(lineText).width
  } else {
    // 回退方案：使用等宽字体估算（Monaco/Menlo 是等宽字体）
    const charWidth = parseFloat(style.fontSize) * 0.6 // 等宽字体字符宽度约为字体大小的 0.6 倍
    textWidth = lineText.length * charWidth
  }
  
  // 计算光标位置的 top（考虑滚动）
  const scrollTop = textarea.scrollTop
  const top = textareaRect.top + paddingTop + (currentLine * lineHeight) - scrollTop + lineHeight + 4
  
  // 计算光标位置的 left（考虑滚动）
  const scrollLeft = textarea.scrollLeft
  const left = textareaRect.left + paddingLeft + textWidth - scrollLeft
  
  return { top, left }
}

// 检查并显示占位符提示
const checkAndShowPlaceholderPopup = () => {
  const textarea = getTextareaElement()
  
  if (!textarea) {
    return
  }
  
  const cursorPos = textarea.selectionStart
  const textBeforeCursor = localSystemPrompt.value.substring(0, cursorPos)
  
  // 检查是否输入了 {{（从光标位置向前查找最近的 {{）
  // 需要找到光标前最近的 {{，且中间没有 }}
  let lastOpenPos = -1
  for (let i = cursorPos - 1; i >= 0; i--) {
    if (i > 0 && textBeforeCursor[i - 1] === '{' && textBeforeCursor[i] === '{') {
      // 找到了 {{
      const textAfterOpen = textBeforeCursor.substring(i + 1)
      // 检查是否已经包含 }}（说明占位符已完成）
      if (!textAfterOpen.includes('}}')) {
        lastOpenPos = i - 1
        break
      }
    }
  }
  
  if (lastOpenPos === -1) {
    // 没有找到有效的 {{，隐藏提示
    showPlaceholderPopup.value = false
    placeholderPrefix.value = ''
    return
  }
  
  // 获取 {{ 之后到光标位置的内容作为前缀
  const textAfterOpen = textBeforeCursor.substring(lastOpenPos + 2)
  
  // 更新前缀
  placeholderPrefix.value = textAfterOpen
  
  // 根据前缀过滤占位符
  const filtered = filteredPlaceholders.value
  
  if (filtered.length > 0) {
    // 有匹配的占位符，显示提示
    // 计算光标位置
    nextTick(() => {
      const position = calculateCursorPosition(textarea)
      popupStyle.value = {
        top: `${position.top}px`,
        left: `${position.left}px`
      }
      showPlaceholderPopup.value = true
      // 重置选中索引为第一个（默认选择第一个）
      selectedPlaceholderIndex.value = 0
    })
  } else {
    // 没有匹配的占位符，隐藏提示
    showPlaceholderPopup.value = false
  }
}

// 处理系统 Prompt 输入
const handlePromptInput = () => {
  // 清除之前的定时器
  if (placeholderPopupTimer) {
    clearTimeout(placeholderPopupTimer)
  }
  
  // 延迟检查，避免频繁触发
  placeholderPopupTimer = setTimeout(() => {
    checkAndShowPlaceholderPopup()
  }, 50)
}

// 插入占位符
const insertPlaceholder = (placeholderName: string) => {
  const textarea = getTextareaElement()
  if (!textarea) {
    return
  }
  
  // 先关闭提示，避免触发 blur 事件
  showPlaceholderPopup.value = false
  placeholderPrefix.value = ''
  selectedPlaceholderIndex.value = 0
  
  // 延迟执行，确保提示框已关闭
  nextTick(() => {
    const cursorPos = textarea.selectionStart
    const textBeforeCursor = localSystemPrompt.value.substring(0, cursorPos)
    const textAfterCursor = localSystemPrompt.value.substring(cursorPos)
    
    // 找到最后一个 {{ 的位置
    const lastOpenPos = textBeforeCursor.lastIndexOf('{{')
    if (lastOpenPos === -1) {
      // 如果没有找到 {{，直接插入完整的占位符
      const placeholder = `{{${placeholderName}}}`
      localSystemPrompt.value = textBeforeCursor + placeholder + textAfterCursor
      // 设置光标位置
      nextTick(() => {
        const newPos = cursorPos + placeholder.length
        textarea.setSelectionRange(newPos, newPos)
        textarea.focus()
      })
    } else {
      // 替换 {{ 到光标位置的内容为完整的占位符
      const beforePlaceholder = textBeforeCursor.substring(0, lastOpenPos)
      const placeholder = `{{${placeholderName}}}`
      localSystemPrompt.value = beforePlaceholder + placeholder + textAfterCursor
      // 设置光标位置
      nextTick(() => {
        const newPos = lastOpenPos + placeholder.length
        textarea.setSelectionRange(newPos, newPos)
        textarea.focus()
      })
    }
  })
}

// 恢复默认 Prompt
const handleResetToDefault = async () => {
  const confirmDialog = DialogPlugin.confirm({
    header: '恢复默认 Prompt',
    body: '确定要恢复为默认 Prompt 吗？当前的自定义 Prompt 将被覆盖。',
    confirmBtn: '确定',
    cancelBtn: '取消',
    onConfirm: async () => {
      try {
        isResettingPrompt.value = true
        
        // 通过设置 system_prompt 为空字符串来获取默认值
        // 后端在 system_prompt 为空时会返回默认值
        const tempConfig: AgentConfig = {
          enabled: isAgentReady.value,
          max_iterations: localMaxIterations.value,
          reflection_enabled: false,
          allowed_tools: localAllowedTools.value,
          temperature: localTemperature.value,
          thinking_model_id: localThinkingModelId.value,
          rerank_model_id: localRerankModelId.value,
          system_prompt: '' // 空字符串表示使用默认
        }
        
        await updateAgentConfig(tempConfig)
        
        // 重新加载配置以获取默认 Prompt 的完整内容
        const res = await getAgentConfig()
        const defaultPrompt = res.data.system_prompt || ''
        
        // 设置为默认 Prompt 的内容
        localSystemPrompt.value = defaultPrompt
        savedSystemPrompt = defaultPrompt
        
        MessagePlugin.success('已恢复为默认 Prompt')
        confirmDialog.hide()
      } catch (error) {
        console.error('恢复默认 Prompt 失败:', error)
        MessagePlugin.error(getErrorMessage(error))
      } finally {
        isResettingPrompt.value = false
      }
    }
  })
}

// 处理系统 Prompt 变化
const handleSystemPromptChange = async (e?: FocusEvent) => {
  // 如果点击的是占位符提示框，不触发保存
  if (e?.relatedTarget) {
    const target = e.relatedTarget as HTMLElement
    if (target.closest('.placeholder-popup-wrapper')) {
      return
    }
  }
  
  // 延迟检查，避免点击占位符时立即触发
  await nextTick()
  
  // 如果占位符提示框还在显示，说明用户点击了占位符，不触发保存
  if (showPlaceholderPopup.value) {
    return
  }
  
  // 隐藏占位符提示
  placeholderPrefix.value = ''
  
  // 如果正在初始化，不触发保存
  if (isInitializing.value) return
  
  // 检查内容是否变化
  if (localSystemPrompt.value === savedSystemPrompt) {
    return // 内容没变，不调用接口
  }
  
  try {
    const config: AgentConfig = {
      enabled: isAgentReady.value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value,
      system_prompt: localSystemPrompt.value
    }
    
    await updateAgentConfig(config)
    savedSystemPrompt = localSystemPrompt.value // 更新已保存的值
    MessagePlugin.success('系统 Prompt 已保存')
  } catch (error) {
    console.error('保存系统 Prompt 失败:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// 监听 Agent 就绪状态变化，同步到 store
watch(isAgentReady, (newValue, oldValue) => {
  if (!isInitializing.value) {
    // 如果配置从"就绪"变为"未就绪"，且 Agent 当前是启用状态，自动关闭
    if (!newValue && oldValue && settingsStore.isAgentEnabled) {
      settingsStore.toggleAgent(false)
      MessagePlugin.warning('Agent 配置不完整，已自动关闭 Agent 模式')
    }
    // 注意：配置从"未就绪"变为"就绪"时，不自动启用（让用户自己决定是否启用）
  }
})
</script>

<style lang="less" scoped>
.agent-settings {
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
    margin: 0 0 20px 0;
    line-height: 1.5;
  }

  .agent-status-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 20px 0;
    border-bottom: 1px solid #e5e7eb;
    margin-top: 8px;

    .status-label {
      flex: 1;
      max-width: 65%;
      padding-right: 24px;

      label {
        font-size: 15px;
        font-weight: 500;
        color: #333333;
        display: block;
        margin-bottom: 4px;
      }
    }

    .status-control {
      flex-shrink: 0;
      min-width: 280px;
      display: flex;
      flex-direction: column;
      align-items: flex-end;
      gap: 8px;

      .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        border-radius: 4px;
        font-size: 14px;
        font-weight: 500;

        &.ready {
          background: #f0fdf4;
          color: #16a34a;
          
          .status-icon {
            color: #16a34a;
            font-size: 16px;
          }
        }

        &:not(.ready) {
          background: #fff7ed;
          color: #ea580c;
          
          .status-icon {
            color: #ea580c;
            font-size: 16px;
          }
        }

        .status-text {
          line-height: 1.4;
        }
      }

      .status-hint {
        font-size: 13px;
        color: #666666;
        text-align: right;
        line-height: 1.5;
        max-width: 280px;
      }

      .status-tip {
        margin: 8px 0 0 0;
        font-size: 12px;
        color: #999999;
        text-align: right;
        line-height: 1.5;
        max-width: 280px;
        display: flex;
        align-items: flex-start;
        gap: 4px;
        justify-content: flex-end;

        .tip-icon {
          font-size: 14px;
          color: #999999;
          flex-shrink: 0;
          margin-top: 2px;
        }
      }
    }
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

  &.vertical {
    flex-direction: column;
    align-items: flex-start;

    .setting-info {
      margin-bottom: 12px;
      max-width: 100%;
    }

    .setting-control.full-width {
      width: 100%;
    }
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;

  label {
    font-size: 15px;
    font-weight: 500;
    color: #333333;
    display: block;
    margin-bottom: 4px;
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
  align-items: center;
}

.slider-with-value {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
  justify-content: flex-end;

  .value-display {
    font-size: 14px;
    font-weight: 500;
    color: #333333;
    min-width: 40px;
    text-align: right;
  }
}

// 模型选择器样式
.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .model-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .add-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .model-name {
    flex: 1;
    font-size: 13px;
  }
  
  &.add {
    .model-name {
      color: #07C05F;
      font-weight: 500;
    }
  }
}

.prompt-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
  width: 100%;
}

.prompt-textarea-wrapper {
  width: 100%;
}

.setting-control.full-width {
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.placeholder-hint {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.6;

  .hint-title {
    font-weight: 500;
    color: #333;
    margin: 0 0 8px 0;
  }

  .placeholder-list {
    margin: 8px 0;
    padding-left: 20px;
    color: #666;

    li {
      margin: 4px 0;

      code {
        background: #fff;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        font-size: 11px;
        color: #e83e8c;
        border: 1px solid #e1e8ed;
      }
    }
  }

  .hint-tip {
    margin: 8px 0 0 0;
    color: #999;
    font-style: italic;
  }
}

.placeholder-popup-wrapper {
  position: fixed;
  z-index: 10001;
  pointer-events: auto;
}

.placeholder-popup {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  max-width: 400px;
  max-height: 300px;
  overflow-y: auto;
  padding: 4px 0;
}

.placeholder-item {
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.2s;

  &:hover,
  &.active {
    background-color: #f5f7fa;
  }

  .placeholder-name {
    font-weight: 500;
    margin-bottom: 4px;

    code {
      background: #f5f7fa;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
      font-size: 12px;
      color: #e83e8c;
    }
  }

  .placeholder-desc {
    font-size: 12px;
    color: #666;
    line-height: 1.4;
  }
}

</style>

