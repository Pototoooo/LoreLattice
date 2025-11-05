<template>
  <div class="kb-settings">
    <div class="section-header">
      <h2>知识库设置</h2>
      <p class="section-description">配置文档处理和索引的默认参数</p>
    </div>

    <div class="settings-group">
      <!-- Chunk 大小 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>默认 Chunk 大小</label>
          <p class="desc">文档分块的默认字符数（100-2000）</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
            <t-slider
              v-model="localChunkSize"
              :min="100"
              :max="2000"
              :step="50"
              :marks="{ 100: '100', 1000: '1000', 2000: '2000' }"
              @change="handleChunkSizeChange"
              style="width: 200px;"
            />
            <span class="value-display">{{ localChunkSize }}</span>
          </div>
        </div>
      </div>

      <!-- Chunk Overlap -->
      <div class="setting-row">
        <div class="setting-info">
          <label>Chunk Overlap</label>
          <p class="desc">相邻文档块之间的重叠字符数（0-500）</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
            <t-slider
              v-model="localChunkOverlap"
              :min="0"
              :max="500"
              :step="20"
              :marks="{ 0: '0', 250: '250', 500: '500' }"
              @change="handleChunkOverlapChange"
              style="width: 200px;"
            />
            <span class="value-display">{{ localChunkOverlap }}</span>
          </div>
        </div>
      </div>

      <!-- 文档语言检测 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>文档语言检测</label>
          <p class="desc">自动检测文档语言以优化分词效果</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localLanguageDetection"
            @change="handleLanguageDetectionChange"
            size="large"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'

// 本地状态
const localChunkSize = ref(500)
const localChunkOverlap = ref(100)
const localLanguageDetection = ref(true)

// 初始化加载
onMounted(() => {
  // 从 localStorage 加载知识库设置
  const savedSettings = localStorage.getItem('WeKnora_kb_settings')
  if (savedSettings) {
    try {
      const parsed = JSON.parse(savedSettings)
      localChunkSize.value = parsed.chunkSize || 500
      localChunkOverlap.value = parsed.chunkOverlap || 100
      localLanguageDetection.value = parsed.languageDetection !== false
    } catch (e) {
      console.error('加载知识库设置失败:', e)
    }
  }
})

// 保存设置到 localStorage
const saveSettings = () => {
  const settings = {
    chunkSize: localChunkSize.value,
    chunkOverlap: localChunkOverlap.value,
    languageDetection: localLanguageDetection.value
  }
  localStorage.setItem('WeKnora_kb_settings', JSON.stringify(settings))
}

// 处理 Chunk 大小变化
const handleChunkSizeChange = () => {
  saveSettings()
  MessagePlugin.success('Chunk 大小已保存')
}

// 处理 Chunk Overlap 变化
const handleChunkOverlapChange = () => {
  // 确保 overlap 不超过 chunk size
  if (localChunkOverlap.value > localChunkSize.value) {
    localChunkOverlap.value = Math.floor(localChunkSize.value / 2)
    MessagePlugin.warning('Overlap 不能超过 Chunk 大小，已自动调整')
  }
  saveSettings()
  MessagePlugin.success('Chunk Overlap 已保存')
}

// 处理语言检测开关
const handleLanguageDetectionChange = () => {
  saveSettings()
  MessagePlugin.success(`文档语言检测已${localLanguageDetection.value ? '启用' : '禁用'}`)
}
</script>

<style lang="less" scoped>
.kb-settings {
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
</style>

