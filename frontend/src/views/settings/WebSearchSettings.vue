<template>
  <div class="websearch-settings">
    <div class="section-header">
      <h2>网络搜索配置</h2>
      <p class="section-description">配置网络搜索功能，在回答问题时可以从互联网获取实时信息补充知识库内容</p>
    </div>

    <div class="settings-group">
      <!-- 搜索引擎提供商 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>搜索引擎提供商</label>
          <p class="desc">选择用于网络搜索的搜索引擎服务</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localProvider"
            :loading="loadingProviders"
            filterable
            placeholder="选择搜索引擎..."
            @change="handleProviderChange"
            @focus="loadProviders"
            style="width: 280px;"
          >
            <t-option
              v-for="provider in providers"
              :key="provider.id"
              :value="provider.id"
              :label="provider.name"
            >
              <div class="provider-option-wrapper">
                <div class="provider-option">
                  <span class="provider-name">{{ provider.name }}</span>
                </div>
              </div>
            </t-option>
          </t-select>
        </div>
      </div>

      <!-- API 密钥 -->
      <div v-if="selectedProvider && selectedProvider.requires_api_key" class="setting-row">
        <div class="setting-info">
          <label>API 密钥</label>
          <p class="desc">输入所选搜索引擎的 API 密钥</p>
        </div>
        <div class="setting-control">
          <t-input
            v-model="localAPIKey"
            type="password"
            placeholder="请输入 API 密钥"
            @change="handleAPIKeyChange"
            style="width: 400px;"
            :show-password="true"
          />
        </div>
      </div>

      <!-- 最大结果数 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>最大结果数</label>
          <p class="desc">每次搜索返回的最大结果数量（1-50）</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
            <t-slider 
              v-model="localMaxResults" 
              :min="1" 
              :max="50" 
              :step="1"
              :marks="{ 1: '1', 10: '10', 20: '20', 30: '30', 40: '40', 50: '50' }"
              @change="handleMaxResultsChange"
              style="width: 200px;"
            />
            <span class="value-display">{{ localMaxResults }}</span>
          </div>
        </div>
      </div>

      <!-- 包含日期 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>包含发布日期</label>
          <p class="desc">在搜索结果中包含内容的发布日期信息</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localIncludeDate"
            @change="handleIncludeDateChange"
          />
        </div>
      </div>

      <!-- 压缩方法 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>压缩方法</label>
          <p class="desc">对搜索结果内容的压缩处理方法</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localCompressionMethod"
            @change="handleCompressionMethodChange"
            style="width: 280px;"
          >
            <t-option value="none" label="无压缩">无压缩</t-option>
            <t-option value="llm_summary" label="LLM 摘要">LLM 摘要</t-option>
          </t-select>
        </div>
      </div>

      <!-- 黑名单 -->
      <div class="setting-row vertical">
        <div class="setting-info">
          <label>URL 黑名单</label>
          <p class="desc">排除特定域名或 URL 的搜索结果，每行一个。支持通配符（*）和正则表达式（以/开头和结尾）</p>
        </div>
        <div class="setting-control">
          <t-textarea
            v-model="localBlacklistText"
            placeholder="例如：&#10;*://*.example.com/*&#10;/example\.(net|org)/"
            :autosize="{ minRows: 4, maxRows: 8 }"
            @change="handleBlacklistChange"
            style="width: 500px;"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { getWebSearchProviders, getTenantWebSearchConfig, updateTenantWebSearchConfig, type WebSearchProviderConfig, type WebSearchConfig } from '@/api/web-search'

// 本地状态
const loadingProviders = ref(false)
const providers = ref<WebSearchProviderConfig[]>([])
const localProvider = ref<string>('')
const localAPIKey = ref<string>('')
const localMaxResults = ref<number>(5)
const localIncludeDate = ref<boolean>(true)
const localCompressionMethod = ref<string>('none')
const localBlacklistText = ref<string>('')

// 计算属性：当前选中的提供商
const selectedProvider = computed(() => {
  return providers.value.find(p => p.id === localProvider.value)
})

// 加载提供商列表
const loadProviders = async () => {
  if (providers.value.length > 0) {
    return // 已加载过
  }
  
  loadingProviders.value = true
  try {
    const response = await getWebSearchProviders()
    // request拦截器已经处理了响应，直接使用data字段
    if (response.data && Array.isArray(response.data)) {
      providers.value = response.data
    }
  } catch (error: any) {
    console.error('Failed to load web search providers:', error)
    MessagePlugin.error('加载搜索引擎列表失败: ' + (error.message || '未知错误'))
  } finally {
    loadingProviders.value = false
  }
}

// 加载租户配置
const loadTenantConfig = async () => {
  try {
    const response = await getTenantWebSearchConfig()
    // request拦截器已经处理了响应，直接使用data字段
    if (response.data) {
      const config = response.data
      localProvider.value = config.provider || ''
      // API key 在响应中被隐藏，如果是 "***"，说明已配置但未返回实际值
      localAPIKey.value = config.api_key === '***' ? '***' : config.api_key || ''
      localMaxResults.value = config.max_results || 5
      localIncludeDate.value = config.include_date !== undefined ? config.include_date : true
      localCompressionMethod.value = config.compression_method || 'none'
      localBlacklistText.value = (config.blacklist || []).join('\n')
    }
  } catch (error: any) {
    console.error('Failed to load tenant web search config:', error)
    // 如果配置不存在，使用默认值（不显示错误）
  }
}

// 保存配置
const saveConfig = async () => {
  try {
    const blacklist = localBlacklistText.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)
    
    const config: WebSearchConfig = {
      provider: localProvider.value,
      api_key: localAPIKey.value,
      max_results: localMaxResults.value,
      include_date: localIncludeDate.value,
      compression_method: localCompressionMethod.value,
      blacklist: blacklist
    }
    
    await updateTenantWebSearchConfig(config)
    MessagePlugin.success('网络搜索配置已保存')
  } catch (error: any) {
    console.error('Failed to save web search config:', error)
    MessagePlugin.error('保存配置失败: ' + (error.message || '未知错误'))
    throw error
  }
}

// 防抖保存
let saveTimer: number | null = null
const debouncedSave = () => {
  if (saveTimer) {
    clearTimeout(saveTimer)
  }
  saveTimer = window.setTimeout(() => {
    saveConfig().catch(() => {
      // 错误已在 saveConfig 中处理
    })
  }, 500)
}

// 处理变化
const handleProviderChange = () => {
  debouncedSave()
}

const handleAPIKeyChange = () => {
  debouncedSave()
}

const handleMaxResultsChange = () => {
  debouncedSave()
}

const handleIncludeDateChange = () => {
  debouncedSave()
}

const handleCompressionMethodChange = () => {
  debouncedSave()
}

const handleBlacklistChange = () => {
  debouncedSave()
}

// 初始化
onMounted(async () => {
  await loadProviders()
  await loadTenantConfig()
})
</script>

<style lang="less" scoped>
.websearch-settings {
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
    gap: 12px;

    .setting-control {
      width: 100%;
      max-width: 100%;
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
  gap: 12px;
}

.value-display {
  min-width: 40px;
  text-align: right;
  font-size: 14px;
  font-weight: 500;
  color: #333333;
}

.provider-option-wrapper {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 2px 0;
}

.provider-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}

.provider-name {
  font-weight: 500;
  font-size: 14px;
  color: #333;
  flex-shrink: 0;
}

.provider-tags {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.provider-desc {
  font-size: 12px;
  color: #999;
  line-height: 1.4;
  margin-top: 2px;
}

/* 修复下拉项描述与条目重叠：让选项支持多行自适应高度 */
:deep(.t-select-option) {
  height: auto;
  align-items: flex-start;
  padding-top: 6px;
  padding-bottom: 6px;
}

:deep(.t-select-option__content) {
  white-space: normal;
}

</style>
<style lang="less">
.t-select__dropdown .t-select-option {
  height: auto;
  align-items: flex-start;
  padding-top: 6px;
  padding-bottom: 6px;
}
.t-select__dropdown .t-select-option__content {
  white-space: normal;
}
.t-select__dropdown .provider-option-wrapper {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 2px 0;
}
</style>

