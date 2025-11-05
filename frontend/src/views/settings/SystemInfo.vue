<template>
  <div class="system-info">
    <div class="section-header">
      <h2>系统信息</h2>
      <p class="section-description">查看系统版本信息和用户账户配置</p>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>正在加载信息...</span>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">重试</t-button>
        </template>
      </t-alert>
    </div>

    <!-- 信息内容 -->
    <div v-else class="settings-group">
      <!-- 系统信息分组 -->
      <div class="info-section-title">系统信息</div>
      
      <!-- 系统版本 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>系统版本</label>
          <p class="desc">当前系统的版本号</p>
        </div>
        <div class="setting-control">
          <span class="info-value">
              {{ systemInfo?.version || '未知' }}
              <span v-if="systemInfo?.commit_id" class="commit-info">
                ({{ systemInfo.commit_id }})
              </span>
          </span>
        </div>
      </div>

      <!-- 构建时间 -->
      <div v-if="systemInfo?.build_time" class="setting-row">
        <div class="setting-info">
          <label>构建时间</label>
          <p class="desc">系统构建的时间</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo.build_time }}</span>
        </div>
      </div>

      <!-- Go版本 -->
      <div v-if="systemInfo?.go_version" class="setting-row">
        <div class="setting-info">
          <label>Go 版本</label>
          <p class="desc">后端使用的 Go 语言版本</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo.go_version }}</span>
        </div>
      </div>

      <!-- 用户信息分组 -->
      <div class="info-section-title">用户信息</div>

      <!-- 用户 ID -->
      <div class="setting-row">
        <div class="setting-info">
          <label>用户 ID</label>
          <p class="desc">您的唯一用户标识</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.id || '-' }}</span>
        </div>
      </div>

      <!-- 用户名 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>用户名</label>
          <p class="desc">您的登录用户名</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.username || '-' }}</span>
        </div>
      </div>

      <!-- 邮箱 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>邮箱</label>
          <p class="desc">您的注册邮箱地址</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.email || '-' }}</span>
        </div>
      </div>

      <!-- 用户创建时间 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>注册时间</label>
          <p class="desc">账户创建的时间</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatDate(userInfo?.created_at) }}</span>
        </div>
      </div>

      <!-- 租户信息分组 -->
      <div class="info-section-title">租户信息</div>

      <!-- 租户 ID -->
      <div class="setting-row">
        <div class="setting-info">
          <label>租户 ID</label>
          <p class="desc">您所属租户的唯一标识</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ tenantInfo?.id || '-' }}</span>
        </div>
      </div>

      <!-- 租户名称 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>租户名称</label>
          <p class="desc">您所属的租户名称</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ tenantInfo?.name || '-' }}</span>
        </div>
      </div>

      <!-- 租户描述 -->
      <div v-if="tenantInfo?.description" class="setting-row">
        <div class="setting-info">
          <label>租户描述</label>
          <p class="desc">租户的详细描述信息</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ tenantInfo.description }}</span>
        </div>
      </div>

      <!-- 租户业务 -->
      <div v-if="tenantInfo?.business" class="setting-row">
        <div class="setting-info">
          <label>租户业务</label>
          <p class="desc">租户所属的业务类型</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ tenantInfo.business }}</span>
        </div>
      </div>

      <!-- 租户状态 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>租户状态</label>
          <p class="desc">租户当前的运行状态</p>
        </div>
        <div class="setting-control">
              <t-tag 
                :theme="getStatusTheme(tenantInfo?.status)" 
                variant="light"
            size="small"
              >
                {{ getStatusText(tenantInfo?.status) }}
              </t-tag>
        </div>
      </div>

      <!-- 租户创建时间 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>租户创建时间</label>
          <p class="desc">租户创建的时间</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatDate(tenantInfo?.created_at) }}</span>
        </div>
      </div>

      <!-- API 信息分组 -->
      <div class="info-section-title">API 信息</div>

      <!-- API Key -->
      <div class="setting-row">
        <div class="setting-info">
          <label>API Key</label>
          <p class="desc">用于 API 调用的密钥，请妥善保管</p>
        </div>
        <div class="setting-control">
          <div class="api-key-control">
          <t-input 
            v-model="displayApiKey" 
            readonly 
            :type="showApiKey ? 'text' : 'password'"
              style="width: 100%; font-family: monospace; font-size: 12px;"
          />
            <t-button 
              size="small" 
              variant="text"
              @click="showApiKey = !showApiKey"
            >
              <t-icon :name="showApiKey ? 'browse-off' : 'browse'" />
            </t-button>
          </div>
        </div>
      </div>

      <!-- 存储配额 -->
      <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
        <div class="setting-info">
          <label>存储配额</label>
          <p class="desc">租户的总存储空间配额</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatBytes(tenantInfo.storage_quota) }}</span>
        </div>
      </div>

      <!-- 已使用存储 -->
      <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
        <div class="setting-info">
          <label>已使用存储</label>
          <p class="desc">已经使用的存储空间</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatBytes(tenantInfo.storage_used || 0) }}</span>
        </div>
      </div>

      <!-- 存储使用率 -->
      <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
        <div class="setting-info">
          <label>存储使用率</label>
          <p class="desc">存储空间的使用百分比</p>
        </div>
        <div class="setting-control">
          <div class="usage-control">
                <span class="usage-text">{{ getUsagePercentage() }}%</span>
                <t-progress 
                  :percentage="getUsagePercentage()" 
                  :show-info="false" 
              size="small"
                  :theme="getUsagePercentage() > 80 ? 'warning' : 'success'"
              style="flex: 1;"
                />
              </div>
        </div>
      </div>

      <!-- 开发文档分组 -->
      <div class="info-section-title">开发文档</div>

      <!-- API 文档 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>API 开发文档</label>
          <p class="desc">查看完整的 API 文档和示例代码</p>
        </div>
        <div class="setting-control">
            <t-button 
              theme="primary" 
            size="small"
              @click="openApiDoc"
            >
                <t-icon name="link" />
            查看文档
            </t-button>
        </div>
    </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getCurrentUser, type TenantInfo, type UserInfo } from '@/api/auth'
import { getSystemInfo, type SystemInfo } from '@/api/system'

// 响应式数据
const tenantInfo = ref<TenantInfo | null>(null)
const userInfo = ref<UserInfo | null>(null)
const systemInfo = ref<SystemInfo | null>(null)
const loading = ref(true)
const error = ref('')
const showApiKey = ref(false)

// 计算属性
const displayApiKey = computed(() => {
  if (!tenantInfo.value?.api_key) return ''
  return tenantInfo.value.api_key
})

// 方法
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    
    // 并行获取用户信息和系统信息
    const [userResponse, systemResponse] = await Promise.all([
      getCurrentUser(),
      getSystemInfo().catch(() => ({ data: null }))
    ])
    
    if (userResponse.success && userResponse.data) {
      userInfo.value = userResponse.data.user
      tenantInfo.value = userResponse.data.tenant
    } else {
      error.value = userResponse.message || '获取用户信息失败'
    }
    
    if (systemResponse.data) {
      systemInfo.value = systemResponse.data
    }
  } catch (err: any) {
    error.value = err.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
}

const openApiDoc = () => {
  window.open('https://github.com/Tencent/WeKnora/blob/main/docs/API.md', '_blank')
}

const getStatusText = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return '活跃'
    case 'inactive':
      return '未激活'
    case 'suspended':
      return '已暂停'
    default:
      return '未知'
  }
}

const getStatusTheme = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'suspended':
      return 'danger'
    default:
      return 'default'
  }
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return '未知'
  
  try {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return '格式错误'
  }
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getUsagePercentage = () => {
  if (!tenantInfo.value?.storage_quota || tenantInfo.value.storage_quota === 0) {
    return 0
  }
  
  const used = tenantInfo.value.storage_used || 0
  const percentage = (used / tenantInfo.value.storage_quota) * 100
  return Math.min(Math.round(percentage * 100) / 100, 100)
}

// 生命周期
onMounted(() => {
  loadInfo()
})
</script>

<style lang="less" scoped>
.system-info {
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

.loading-inline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px 0;
  justify-content: center;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.error-inline {
  padding: 20px 0;
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

  .info-value {
    font-size: 14px;
    color: var(--td-text-color-primary);
    text-align: right;
    word-break: break-word;

    .commit-info {
      color: var(--td-text-color-placeholder);
      font-size: 12px;
      margin-left: 6px;
    }
  }
}

.api-key-control {
  width: 100%;
  display: flex;
  gap: 8px;
  align-items: center;
}

.usage-control {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;

  .usage-text {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    min-width: 50px;
    text-align: right;
  }
}

.info-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin-top: 24px;
  margin-bottom: 12px;

  &:first-child {
    margin-top: 0;
  }
}
</style>
