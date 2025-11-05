<template>
  <div class="general-settings">
    <div class="section-header">
      <h2>常规设置</h2>
      <p class="section-description">配置语言、外观等基础选项</p>
    </div>

    <div class="settings-group">
      <!-- 语言选择 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>语言</label>
          <p class="desc">选择界面显示语言</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localLanguage"
            placeholder="选择语言"
            @change="handleLanguageChange"
            style="width: 280px;"
          >
            <t-option value="zh-CN" label="中文">中文</t-option>
            <t-option value="en-US" label="English">English</t-option>
          </t-select>
        </div>
      </div>

      <!-- 外观主题 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>外观</label>
          <p class="desc">选择界面主题样式</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localTheme"
            placeholder="选择主题"
            @change="handleThemeChange"
            style="width: 280px;"
          >
            <t-option value="light" label="明亮">明亮</t-option>
            <t-option value="dark" label="暗色">暗色</t-option>
          </t-select>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useUIStore } from '@/stores/ui'

const uiStore = useUIStore()

// 本地状态
const localLanguage = ref('zh-CN')
const localTheme = ref('light')

// 初始化加载
onMounted(() => {
  // 从 localStorage 加载常规设置
  const savedSettings = localStorage.getItem('WeKnora_general_settings')
  if (savedSettings) {
    try {
      const parsed = JSON.parse(savedSettings)
      localLanguage.value = parsed.language || 'zh-CN'
      localTheme.value = parsed.theme || 'light'
    } catch (e) {
      console.error('加载常规设置失败:', e)
    }
  }
  // 同步 store 的主题状态
  localTheme.value = uiStore.theme
})

// 监听 store 的主题变化，同步到本地状态
watch(() => uiStore.theme, (newTheme) => {
  localTheme.value = newTheme
})

// 保存设置到 localStorage
const saveSettings = () => {
  const settings = {
    language: localLanguage.value,
    theme: localTheme.value
  }
  localStorage.setItem('WeKnora_general_settings', JSON.stringify(settings))
}

// 处理语言变化
const handleLanguageChange = () => {
  saveSettings()
  MessagePlugin.success('语言设置已保存')
  // TODO: 实际的语言切换逻辑可后续添加
}

// 处理主题变化
const handleThemeChange = () => {
  uiStore.setTheme(localTheme.value as 'light' | 'dark')
  MessagePlugin.success('主题设置已保存')
}
</script>

<style lang="less" scoped>
.general-settings {
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
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}
</style>

