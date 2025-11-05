<template>
  <div class="kb-basic-info">
    <div class="section-header">
      <h2>基本信息</h2>
      <p class="section-description">配置知识库的基本信息</p>
    </div>

    <div class="settings-group">
      <!-- 知识库名称 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>知识库名称 <span class="required">*</span></label>
          <p class="desc">知识库的显示名称（最多50个字符）</p>
        </div>
        <div class="setting-control">
          <t-input
            v-model="localName"
            placeholder="请输入知识库名称"
            maxlength="50"
            show-word-limit
            @blur="handleNameChange"
            style="width: 280px;"
          />
        </div>
      </div>

      <!-- 知识库描述 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>知识库描述</label>
          <p class="desc">知识库的详细描述信息（最多200个字符）</p>
        </div>
        <div class="setting-control">
          <t-textarea
            v-model="localDescription"
            placeholder="请输入知识库描述"
            maxlength="200"
            show-word-limit
            :autosize="{ minRows: 3, maxRows: 6 }"
            @blur="handleDescriptionChange"
            style="width: 280px;"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { updateKnowledgeBase } from '@/api/knowledge-base'

interface Props {
  kbId: string
  name: string
  description?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:name': [value: string]
  'update:description': [value: string]
}>()

const localName = ref(props.name)
const localDescription = ref(props.description || '')
const saving = ref(false)

// 监听props变化
watch(() => props.name, (newVal) => {
  localName.value = newVal
})

watch(() => props.description, (newVal) => {
  localDescription.value = newVal || ''
})

// 保存基本信息
const saveBasicInfo = async () => {
  if (saving.value) return
  
  if (!localName.value.trim()) {
    MessagePlugin.warning('知识库名称不能为空')
    return
  }

  saving.value = true
  try {
    await updateKnowledgeBase(props.kbId, {
      name: localName.value,
      description: localDescription.value,
      config: {} // 这里只更新基本信息，不更新配置
    })
    
    emit('update:name', localName.value)
    emit('update:description', localDescription.value)
    MessagePlugin.success('保存成功')
  } catch (error) {
    console.error('保存知识库基本信息失败:', error)
    MessagePlugin.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 处理名称变化
const handleNameChange = () => {
  if (localName.value !== props.name) {
    saveBasicInfo()
  }
}

// 处理描述变化
const handleDescriptionChange = () => {
  if (localDescription.value !== props.description) {
    saveBasicInfo()
  }
}
</script>

<style lang="less" scoped>
.kb-basic-info {
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

    .required {
      color: var(--td-error-color);
      margin-left: 2px;
    }
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

