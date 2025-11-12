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
          <t-radio value="stdio">Stdio</t-radio>
        </t-radio-group>
      </t-form-item>

      <!-- URL for SSE/HTTP Streamable -->
      <t-form-item 
        v-if="formData.transport_type !== 'stdio'" 
        label="服务 URL" 
        name="url"
      >
        <t-input v-model="formData.url" placeholder="https://example.com/mcp" />
      </t-form-item>

      <!-- Stdio Config -->
      <template v-if="formData.transport_type === 'stdio'">
        <t-form-item label="命令" name="stdio_config.command">
          <t-radio-group v-model="formData.stdio_config.command">
            <t-radio value="uvx">uvx</t-radio>
            <t-radio value="npx">npx</t-radio>
          </t-radio-group>
        </t-form-item>

        <t-form-item label="参数" name="stdio_config.args">
          <div class="args-input-container">
            <div 
              v-for="(arg, index) in formData.stdio_config.args" 
              :key="index" 
              class="arg-item"
            >
              <t-input 
                v-model="formData.stdio_config.args[index]" 
                :placeholder="`参数 ${index + 1}`"
                class="arg-input"
              />
              <t-button 
                variant="text" 
                theme="danger" 
                @click="removeArg(index)"
                :disabled="formData.stdio_config.args.length === 1"
              >
                <template #icon><t-icon name="delete" /></template>
              </t-button>
            </div>
            <t-button 
              variant="outline" 
              size="small" 
              @click="addArg"
              class="add-arg-btn"
            >
              <template #icon><t-icon name="add" /></template>
              添加参数
            </t-button>
          </div>
        </t-form-item>

        <t-form-item label="环境变量">
          <div class="env-vars-container">
            <div 
              v-for="(value, key, index) in formData.env_vars" 
              :key="index" 
              class="env-var-item"
            >
              <t-input 
                v-model="envVarKeys[index]" 
                placeholder="变量名"
                class="env-key-input"
                @blur="updateEnvVarKey(index, envVarKeys[index])"
              />
              <t-input 
                v-model="formData.env_vars[key]" 
                placeholder="变量值"
                type="password"
                class="env-value-input"
              />
              <t-button 
                variant="text" 
                theme="danger" 
                @click="removeEnvVar(key)"
              >
                <template #icon><t-icon name="delete" /></template>
              </t-button>
            </div>
            <t-button 
              variant="outline" 
              size="small" 
              @click="addEnvVar"
              class="add-env-var-btn"
            >
              <template #icon><t-icon name="add" /></template>
              添加环境变量
            </t-button>
          </div>
        </t-form-item>
      </template>

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
  transport_type: 'sse' as 'sse' | 'http-streamable' | 'stdio',
  url: '',
  stdio_config: {
    command: 'uvx' as 'uvx' | 'npx',
    args: ['']
  },
  env_vars: {} as Record<string, string>,
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

// Track env var keys separately for easier editing
const envVarKeys = ref<string[]>([])

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入服务名称', type: 'error' }],
  transport_type: [{ required: true, message: '请选择传输类型', type: 'error' }],
  url: [
    { 
      validator: (val: string) => {
        if (formData.value.transport_type !== 'stdio') {
          if (!val || val.trim() === '') {
            return { result: false, message: '请输入服务 URL', type: 'error' }
          }
          // Basic URL validation
          try {
            new URL(val)
            return { result: true }
          } catch {
            return { result: false, message: '请输入有效的 URL', type: 'error' }
          }
        }
        return { result: true }
      }
    }
  ],
  'stdio_config.command': [
    {
      validator: (val: string) => {
        if (formData.value.transport_type === 'stdio') {
          if (!val || (val !== 'uvx' && val !== 'npx')) {
            return { result: false, message: '请选择命令 (uvx 或 npx)', type: 'error' }
          }
        }
        return { result: true }
      }
    }
  ],
  'stdio_config.args': [
    {
      validator: (val: string[]) => {
        if (formData.value.transport_type === 'stdio') {
          if (!val || val.length === 0 || val.every(arg => !arg || arg.trim() === '')) {
            return { result: false, message: '请至少输入一个参数', type: 'error' }
          }
        }
        return { result: true }
      }
    }
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
    stdio_config: {
      command: 'uvx',
      args: ['']
    },
    env_vars: {},
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
  envVarKeys.value = []
  formRef.value?.clearValidate()
}

// Watch transport_type to reset related fields
watch(
  () => formData.value.transport_type,
  (newType) => {
    if (newType === 'stdio') {
      formData.value.url = ''
      if (!formData.value.stdio_config || formData.value.stdio_config.args.length === 0) {
        formData.value.stdio_config = {
          command: 'uvx',
          args: ['']
        }
      }
    } else {
      formData.value.stdio_config = {
        command: 'uvx',
        args: ['']
      }
      formData.value.env_vars = {}
      envVarKeys.value = []
    }
    formRef.value?.clearValidate()
  }
)

// Args management
const addArg = () => {
  formData.value.stdio_config.args.push('')
}

const removeArg = (index: number) => {
  if (formData.value.stdio_config.args.length > 1) {
    formData.value.stdio_config.args.splice(index, 1)
  }
}

// Env vars management
const addEnvVar = () => {
  const key = `VAR_${Date.now()}`
  formData.value.env_vars[key] = ''
  envVarKeys.value.push(key)
}

const removeEnvVar = (key: string) => {
  delete formData.value.env_vars[key]
  const index = envVarKeys.value.indexOf(key)
  if (index > -1) {
    envVarKeys.value.splice(index, 1)
  }
}

const updateEnvVarKey = (index: number, newKey: string) => {
  const oldKey = envVarKeys.value[index]
  if (oldKey && oldKey !== newKey && formData.value.env_vars[oldKey] !== undefined) {
    const value = formData.value.env_vars[oldKey]
    delete formData.value.env_vars[oldKey]
    if (newKey && newKey.trim() !== '') {
      formData.value.env_vars[newKey] = value
      envVarKeys.value[index] = newKey
    } else {
      envVarKeys.value.splice(index, 1)
    }
  }
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
        stdio_config: service.stdio_config || {
          command: 'uvx',
          args: ['']
        },
        env_vars: service.env_vars || {},
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
      // Initialize env var keys
      envVarKeys.value = Object.keys(formData.value.env_vars)
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
      auth_config: {
        api_key: formData.value.auth_config.api_key || undefined,
        token: formData.value.auth_config.token || undefined
      },
      advanced_config: formData.value.advanced_config
    }

    // Add URL or stdio_config based on transport type
    if (formData.value.transport_type === 'stdio') {
      // Filter out empty args
      const args = formData.value.stdio_config.args.filter(arg => arg && arg.trim() !== '')
      data.stdio_config = {
        command: formData.value.stdio_config.command,
        args: args.length > 0 ? args : undefined
      }
      // Filter out empty env vars
      const envVars: Record<string, string> = {}
      for (const [key, value] of Object.entries(formData.value.env_vars)) {
        if (key && key.trim() !== '' && value && value.trim() !== '') {
          envVars[key] = value
        }
      }
      data.env_vars = Object.keys(envVars).length > 0 ? envVars : undefined
    } else {
      data.url = formData.value.url || undefined
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

<style scoped lang="less">
.args-input-container {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .arg-item {
    display: flex;
    gap: 8px;
    align-items: center;

    .arg-input {
      flex: 1;
    }
  }

  .add-arg-btn {
    align-self: flex-start;
  }
}

.env-vars-container {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .env-var-item {
    display: flex;
    gap: 8px;
    align-items: center;

    .env-key-input {
      width: 150px;
    }

    .env-value-input {
      flex: 1;
    }
  }

  .add-env-var-btn {
    align-self: flex-start;
  }
}
</style>

