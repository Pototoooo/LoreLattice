<template>
  <t-dialog
    v-model:visible="dialogVisible"
    :header="mode === 'add' ? '添加 MCP 服务' : '编辑 MCP 服务'"
    width="700px"
    :on-confirm="handleSubmit"
    :on-cancel="handleClose"
    :confirm-btn="{ content: '保存', loading: submitting }"
  >
    <t-form
      ref="formRef"
      :data="formData"
      :rules="rules"
      label-width="120px"
    >
      <t-form-item label="服务名称" name="name">
        <t-input v-model="formData.name" placeholder="请输入服务名称" />
      </t-form-item>

      <t-form-item label="描述" name="description">
        <t-textarea
          v-model="formData.description"
          :autosize="{ minRows: 3, maxRows: 5 }"
          placeholder="请输入服务描述"
        />
      </t-form-item>

      <t-form-item label="传输类型" name="transport_type">
        <t-radio-group v-model="formData.transport_type">
          <t-radio value="sse">SSE (Server-Sent Events)</t-radio>
          <t-radio value="http-streamable">HTTP Streamable</t-radio>
        </t-radio-group>
      </t-form-item>

      <t-form-item label="服务 URL" name="url">
        <t-input v-model="formData.url" placeholder="https://example.com/mcp" />
      </t-form-item>

      <t-form-item label="启用服务" name="enabled">
        <t-switch v-model="formData.enabled" />
      </t-form-item>

      <!-- Authentication Config -->
      <t-collapse :default-value="[]">
        <t-collapse-panel header="认证配置" value="auth">
          <t-form-item label="API Key">
            <t-input
              v-model="formData.auth_config.api_key"
              type="password"
              placeholder="可选"
            />
          </t-form-item>
          <t-form-item label="Bearer Token">
            <t-input
              v-model="formData.auth_config.token"
              type="password"
              placeholder="可选"
            />
          </t-form-item>
        </t-collapse-panel>

        <!-- Advanced Config -->
        <t-collapse-panel header="高级配置" value="advanced">
          <t-form-item label="超时时间(秒)">
            <t-input-number
              v-model="formData.advanced_config.timeout"
              :min="1"
              :max="300"
              placeholder="30"
            />
          </t-form-item>
          <t-form-item label="重试次数">
            <t-input-number
              v-model="formData.advanced_config.retry_count"
              :min="0"
              :max="10"
              placeholder="3"
            />
          </t-form-item>
          <t-form-item label="重试延迟(秒)">
            <t-input-number
              v-model="formData.advanced_config.retry_delay"
              :min="0"
              :max="60"
              placeholder="1"
            />
          </t-form-item>
        </t-collapse-panel>
      </t-collapse>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import {
  createMCPService,
  updateMCPService,
  type MCPService
} from '@/api/mcp-service'

interface Props {
  visible: boolean
  service: MCPService | null
  mode: 'add' | 'edit'
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formRef = ref<FormInstanceFunctions>()
const submitting = ref(false)

const formData = ref({
  name: '',
  description: '',
  enabled: true,
  transport_type: 'sse' as 'sse' | 'http-streamable',
  url: '',
  auth_config: {
    api_key: '',
    token: ''
  },
  advanced_config: {
    timeout: 30,
    retry_count: 3,
    retry_delay: 1
  }
})

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入服务名称', type: 'error' }],
  transport_type: [{ required: true, message: '请选择传输类型', type: 'error' }],
  url: [
    { required: true, message: '请输入服务 URL', type: 'error' },
    { url: true, message: '请输入有效的 URL', type: 'error' }
  ]
}

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

// Reset form function - defined before watch to avoid hoisting issues
const resetForm = () => {
  formData.value = {
    name: '',
    description: '',
    enabled: true,
    transport_type: 'sse',
    url: '',
    auth_config: {
      api_key: '',
      token: ''
    },
    advanced_config: {
      timeout: 30,
      retry_count: 3,
      retry_delay: 1
    }
  }
  formRef.value?.clearValidate()
}

// Watch service prop to initialize form
watch(
  () => props.service,
  (service) => {
    if (service) {
      formData.value = {
        name: service.name || '',
        description: service.description || '',
        enabled: service.enabled ?? true,
        transport_type: service.transport_type || 'sse',
        url: service.url || '',
        auth_config: {
          api_key: service.auth_config?.api_key || '',
          token: service.auth_config?.token || ''
        },
        advanced_config: {
          timeout: service.advanced_config?.timeout || 30,
          retry_count: service.advanced_config?.retry_count || 3,
          retry_delay: service.advanced_config?.retry_delay || 1
        }
      }
    } else {
      resetForm()
    }
  },
  { immediate: true }
)

// Handle submit
const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (!valid) return

  submitting.value = true
  try {
    const data: Partial<MCPService> = {
      name: formData.value.name,
      description: formData.value.description,
      enabled: formData.value.enabled,
      transport_type: formData.value.transport_type,
      url: formData.value.url,
      auth_config: {
        api_key: formData.value.auth_config.api_key || undefined,
        token: formData.value.auth_config.token || undefined
      },
      advanced_config: formData.value.advanced_config
    }

    if (props.mode === 'add') {
      await createMCPService(data)
      MessagePlugin.success('MCP 服务已创建')
    } else {
      await updateMCPService(props.service!.id, data)
      MessagePlugin.success('MCP 服务已更新')
    }

    emit('success')
  } catch (error) {
    MessagePlugin.error(
      props.mode === 'add' ? '创建 MCP 服务失败' : '更新 MCP 服务失败'
    )
    console.error('Failed to save MCP service:', error)
  } finally {
    submitting.value = false
  }
}

// Handle close
const handleClose = () => {
  dialogVisible.value = false
}
</script>

