<template>
  <div class="mcp-settings">
    <div class="section-header">
      <h2>MCP 服务管理</h2>
      <p class="section-description">
        管理外部 MCP (Model Context Protocol) 服务，在 Agent 模式下调用外部工具和资源
      </p>
    </div>

    <div v-if="loading" class="loading-container">
      <t-loading text="加载中..." />
    </div>

    <div v-else class="services-container">
      <div class="services-header">
        <div class="header-info">
          <h3>已配置的服务</h3>
          <p>管理和测试 MCP 服务连接</p>
        </div>
        <t-button size="small" theme="primary" @click="handleAdd">
          <template #icon><t-icon name="add" /></template>
          添加服务
        </t-button>
      </div>

      <div v-if="services.length === 0" class="empty-state">
        <t-empty description="暂无 MCP 服务">
          <t-button theme="primary" @click="handleAdd">添加第一个 MCP 服务</t-button>
        </t-empty>
      </div>

      <div v-else class="services-list">
        <div v-for="service in services" :key="service.id" class="service-card">
          <div class="service-info">
            <div class="service-header">
              <div class="service-name">
                {{ service.name }}
                <t-tag 
                  :theme="service.transport_type === 'sse' ? 'success' : 'primary'" 
                  size="small"
                  variant="light"
                >
                  {{ service.transport_type === 'sse' ? 'SSE' : 'HTTP Streamable' }}
                </t-tag>
              </div>
              <div class="service-controls">
                <t-switch 
                  v-model="service.enabled" 
                  @change="() => handleToggleEnabled(service)"
                  size="large"
                />
                <t-dropdown 
                  :options="getServiceOptions(service)" 
                  @click="(data: any) => handleMenuAction(data, service)"
                  placement="bottom-right"
                  :disabled="testing"
                >
                  <t-button variant="text" shape="square" size="small" class="more-btn" :disabled="testing">
                    <t-icon name="more" />
                  </t-button>
                </t-dropdown>
              </div>
            </div>
            <div v-if="service.description" class="service-description">
              {{ service.description }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add/Edit Dialog -->
    <McpServiceDialog
      v-model:visible="dialogVisible"
      :service="currentService"
      :mode="dialogMode"
      @success="handleDialogSuccess"
    />

    <!-- Test Result Dialog -->
    <McpTestResult
      v-model:visible="testDialogVisible"
      :result="testResult"
      :service-name="testingServiceName"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  listMCPServices,
  updateMCPService,
  deleteMCPService,
  testMCPService,
  type MCPService,
  type MCPTestResult
} from '@/api/mcp-service'
import McpServiceDialog from './components/McpServiceDialog.vue'
import McpTestResult from './components/McpTestResult.vue'

const services = ref<MCPService[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'add' | 'edit'>('add')
const currentService = ref<MCPService | null>(null)
const testDialogVisible = ref(false)
const testResult = ref<MCPTestResult | null>(null)
const testingServiceName = ref('')
const testing = ref(false)

// Load MCP services
const loadServices = async () => {
  loading.value = true
  try {
    services.value = await listMCPServices()
  } catch (error) {
    MessagePlugin.error('加载 MCP 服务列表失败')
    console.error('Failed to load MCP services:', error)
  } finally {
    loading.value = false
  }
}

// Handle add button click
const handleAdd = () => {
  currentService.value = null
  dialogMode.value = 'add'
  dialogVisible.value = true
}

// Handle edit button click
const handleEdit = (service: MCPService) => {
  currentService.value = { ...service }
  dialogMode.value = 'edit'
  dialogVisible.value = true
}

// Handle dialog success
const handleDialogSuccess = () => {
  dialogVisible.value = false
  loadServices()
}

// Handle toggle enabled/disabled
const handleToggleEnabled = async (service: MCPService) => {
  if (!service || !service.id) return
  
  const originalState = service.enabled
  try {
    await updateMCPService(service.id, { enabled: service.enabled })
    MessagePlugin.success(`已${service.enabled ? '启用' : '禁用'} MCP 服务`)
  } catch (error) {
    // Revert on error
    service.enabled = originalState
    MessagePlugin.error('更新 MCP 服务状态失败')
    console.error('Failed to update MCP service:', error)
  }
}

// Handle test button click
const handleTest = async (service: MCPService) => {
  if (!service || !service.id) return
  
  testingServiceName.value = service.name || 'MCP 服务'
  testing.value = true
  
  // 显示测试开始提示
  MessagePlugin.info({
    content: `正在测试 ${service.name}...`,
    duration: 0, // 不自动关闭
    closeBtn: false
  })
  
  try {
    const result = await testMCPService(service.id)
    
    console.log('Test result received:', result)
    
    // 关闭所有消息提示
    MessagePlugin.closeAll()
    
    // 检查结果是否存在
    if (!result) {
      // 即使没有结果，也显示错误对话框
      testResult.value = {
        success: false,
        message: '测试失败：未收到服务器响应'
      }
      testDialogVisible.value = true
      return
    }
    
    // 设置测试结果
    testResult.value = result
    
    // 显示详细结果对话框
    console.log('Opening test dialog, result:', testResult.value)
    testDialogVisible.value = true
  } catch (error: any) {
    // 关闭所有消息提示
    MessagePlugin.closeAll()
    
    // 显示错误信息
    const errorMessage = error?.response?.data?.error?.message || error?.message || '测试 MCP 服务失败'
    console.error('Failed to test MCP service:', error)
    
    // 即使出错也显示结果对话框，显示错误信息
    testResult.value = {
      success: false,
      message: errorMessage
    }
    testDialogVisible.value = true
  } finally {
    // 确保关闭 loading
    testing.value = false
  }
}

// Handle delete button click
const handleDelete = async (service: MCPService) => {
  if (!service || !service.id) return
  
  const confirmDialog = DialogPlugin.confirm({
    header: '确认删除',
    body: `确定要删除 MCP 服务"${service.name || '未命名'}"吗？此操作无法撤销。`,
    confirmBtn: '删除',
    cancelBtn: '取消',
    theme: 'warning',
    onConfirm: async () => {
      try {
        await deleteMCPService(service.id)
        MessagePlugin.success('MCP 服务已删除')
        confirmDialog.hide()
        loadServices()
      } catch (error) {
        MessagePlugin.error('删除 MCP 服务失败')
        console.error('Failed to delete MCP service:', error)
      }
    }
  })
}

// Get service options for dropdown menu
const getServiceOptions = (service: MCPService) => {
  return [
    {
      content: '测试连接',
      value: `test-${service.id}`
    },
    {
      content: '编辑',
      value: `edit-${service.id}`
    },
    {
      content: '删除',
      value: `delete-${service.id}`,
      theme: 'error'
    }
  ]
}

// Handle menu action
const handleMenuAction = (data: { value: string }, service: MCPService) => {
  const value = data.value
  
  if (value.startsWith('test-')) {
    handleTest(service)
  } else if (value.startsWith('edit-')) {
    handleEdit(service)
  } else if (value.startsWith('delete-')) {
    handleDelete(service)
  }
}

onMounted(() => {
  loadServices()
})
</script>

<style scoped lang="less">
.mcp-settings {
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

.loading-container {
  padding: 40px 0;
  text-align: center;
}

.services-container {
  margin-top: 16px;
}

.services-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e5e7eb;

  .header-info {
    flex: 1;

    h3 {
      font-size: 15px;
      font-weight: 500;
      color: #333333;
      margin: 0 0 4px 0;
    }

    p {
      font-size: 13px;
      color: #999999;
      margin: 0;
      line-height: 1.5;
    }
  }
}

.empty-state {
  padding: 80px 0;
  text-align: center;

  :deep(.t-empty__description) {
    font-size: 14px;
    color: #999999;
    margin-bottom: 16px;
  }
}

.services-list {
  display: flex;
  flex-direction: column;
  gap: 0;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 16px;
  background: #fafafa;
}

.service-card {
  padding: 12px 0;
  border-bottom: 1px solid #e5e7eb;
  transition: all 0.2s;

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  &:first-child {
    padding-top: 0;
  }
}

.service-info {
  .service-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;

    .service-name {
      font-size: 15px;
      font-weight: 500;
      color: #333333;
      display: flex;
      align-items: center;
      gap: 8px;
      flex: 1;
    }

    .service-controls {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-shrink: 0;

      .more-btn {
        color: #999999;
        padding: 4px;
        transition: all 0.2s;

        &:hover {
          background: #f5f7fa;
          color: #333333;
        }
      }
    }
  }

  .service-description {
    font-size: 13px;
    color: #666666;
    margin-bottom: 8px;
    line-height: 1.5;
  }

  .service-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
    color: #999999;

    .meta-item {
      display: flex;
      align-items: center;
      gap: 4px;

      .meta-icon {
        font-size: 12px;
      }
    }
  }
}
</style>

