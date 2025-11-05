<template>
  <div class="agent-settings">
    <div class="section-header">
      <h2>Agent 配置</h2>
      <p class="section-description">配置 AI Agent 的默认行为和参数，这些设置将应用于所有启用 Agent 模式的对话</p>
    </div>

    <div class="settings-group">
      <!-- 启用 Agent 模式 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>启用 Agent 模式</label>
          <p class="desc">启用后，AI 将能够使用工具进行多步推理和跨知识库搜索</p>
        </div>
        <div class="setting-control">
        <t-switch 
          v-model="localAgentEnabled" 
          @change="handleAgentToggle"
          size="large"
        />
      </div>
      </div>

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
            @change="handleMaxIterationsChange"
            :disabled="!localAgentEnabled"
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
          @focus="loadChatModels"
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
          @focus="loadRerankModels"
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
            :disabled="!localAgentEnabled"
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
            :disabled="!localAgentEnabled"
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import { MessagePlugin } from 'tdesign-vue-next'
import { listModels, type ModelConfig } from '@/api/model'
import { getAgentConfig, updateAgentConfig, type AgentConfig, type ToolDefinition } from '@/api/system'

const settingsStore = useSettingsStore()
const router = useRouter()

// 本地状态
const localAgentEnabled = ref(false)
const localMaxIterations = ref(5)
const localTemperature = ref(0.7)
const localThinkingModelId = ref('')
const localRerankModelId = ref('')
const localAllowedTools = ref<string[]>([])

// 模型列表状态
const chatModels = ref<ModelConfig[]>([])
const rerankModels = ref<ModelConfig[]>([])
const loadingModels = ref(false)

// 可用工具列表
const availableTools = ref<ToolDefinition[]>([])

// 配置加载状态
const loadingConfig = ref(false)

// 初始化加载
onMounted(async () => {
  loadingConfig.value = true
  try {
    // 从后台加载配置
    const res = await getAgentConfig()
    const config = res.data
    
    // 更新本地状态
    localAgentEnabled.value = config.enabled
    localMaxIterations.value = config.max_iterations
    localTemperature.value = config.temperature
    localThinkingModelId.value = config.thinking_model_id
    localRerankModelId.value = config.rerank_model_id
    localAllowedTools.value = config.allowed_tools || []
    availableTools.value = config.available_tools || []
    
    // 如果配置了思考模型，立即加载模型列表以显示模型名称
    if (config.thinking_model_id) {
      await loadChatModels()
    }
    
    // 如果配置了 rerank 模型，立即加载模型列表以显示模型名称
    if (config.rerank_model_id) {
      await loadRerankModels()
    }
    
    // 同步到store
    settingsStore.toggleAgent(config.enabled)
    settingsStore.updateAgentConfig({
      maxIterations: config.max_iterations,
      temperature: config.temperature,
      thinkingModelId: config.thinking_model_id,
      rerankModelId: config.rerank_model_id
    })
  } catch (error) {
    console.error('加载Agent配置失败:', error)
    MessagePlugin.error('加载Agent配置失败')
    
    // 失败时从store加载
    localAgentEnabled.value = settingsStore.isAgentEnabled
    localMaxIterations.value = settingsStore.agentConfig.maxIterations
    localTemperature.value = settingsStore.agentConfig.temperature
    localThinkingModelId.value = settingsStore.agentConfig.thinkingModelId
    localRerankModelId.value = settingsStore.agentConfig.rerankModelId
  } finally {
    loadingConfig.value = false
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

// 处理 Agent 开关
const handleAgentToggle = async (value: boolean) => {
  try {
    // 构建完整配置
    const config: AgentConfig = {
      enabled: value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value
    }
    
    await updateAgentConfig(config)
    settingsStore.toggleAgent(value)
    MessagePlugin.success(value ? 'Agent 模式已启用' : 'Agent 模式已禁用')
  } catch (error) {
    console.error('保存Agent配置失败:', error)
    MessagePlugin.error(getErrorMessage(error))
    // 回滚
    localAgentEnabled.value = !value
  }
}

// 处理最大迭代次数变化
const handleMaxIterationsChange = async (value: number) => {
  try {
    const config: AgentConfig = {
      enabled: localAgentEnabled.value,
      max_iterations: value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value
    }
    
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ maxIterations: value })
    MessagePlugin.success('最大迭代次数已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// 加载对话模型列表
const loadChatModels = async () => {
  if (chatModels.value.length > 0) return // 已经加载过
  
  loadingModels.value = true
  try {
    const allModels = await listModels()
    // 只获取对话模型（KnowledgeQA 类型）
    chatModels.value = allModels.filter(m => m.type === 'KnowledgeQA')
  } catch (error) {
    MessagePlugin.error('加载模型列表失败')
  } finally {
    loadingModels.value = false
  }
}

// 加载 Rerank 模型列表
const loadRerankModels = async () => {
  if (rerankModels.value.length > 0) return // 已经加载过
  
  loadingModels.value = true
  try {
    const allModels = await listModels()
    rerankModels.value = allModels.filter(m => m.type === 'Rerank')
  } catch (error) {
    console.error('加载 Rerank 模型列表失败:', error)
    MessagePlugin.error('加载 Rerank 模型列表失败')
  } finally {
    loadingModels.value = false
  }
}

// 处理思考模型变化
const handleThinkingModelChange = async (value: string) => {
  // 如果选择添加新模型，跳转到模型配置页
  if (value === '__add_model__') {
    router.push('/settings?section=models')
    return
  }
  
  try {
    const config: AgentConfig = {
      enabled: localAgentEnabled.value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: value,
      rerank_model_id: localRerankModelId.value
    }
    
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ thinkingModelId: localThinkingModelId.value })
    MessagePlugin.success('思考模型已保存')
  } catch (error) {
    console.error('保存失败:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// 监听模型选择，处理"添加模型"跳转
// 处理 Rerank 模型变化
const handleRerankModelChange = async (value: string) => {
  // 如果选择添加新模型，跳转到模型配置页
  if (value === '__add_model__') {
    router.push('/settings?section=models&subsection=rerank')
    return
  }
  
  try {
    const config: AgentConfig = {
      enabled: localAgentEnabled.value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: value
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
  try {
    const config: AgentConfig = {
      enabled: localAgentEnabled.value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: localAllowedTools.value,
      temperature: value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value
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
  try {
    const config: AgentConfig = {
      enabled: localAgentEnabled.value,
      max_iterations: localMaxIterations.value,
      reflection_enabled: false,
      allowed_tools: value,
      temperature: localTemperature.value,
      thinking_model_id: localThinkingModelId.value,
      rerank_model_id: localRerankModelId.value
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
  padding-right: 24px;

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

</style>

