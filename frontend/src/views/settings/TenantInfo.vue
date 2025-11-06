<template>
  <div class="tenant-info">
    <div class="section-header">
      <h2>租户信息</h2>
      <p class="section-description">查看租户的详细配置信息</p>
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

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getCurrentUser, type TenantInfo } from '@/api/auth'

// 响应式数据
const tenantInfo = ref<TenantInfo | null>(null)
const loading = ref(true)
const error = ref('')

// 方法
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    
    const userResponse = await getCurrentUser()
    
    if (userResponse.success && userResponse.data) {
      tenantInfo.value = userResponse.data.tenant
    } else {
      error.value = userResponse.message || '获取租户信息失败'
    }
  } catch (err: any) {
    error.value = err.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
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
.tenant-info {
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

.loading-inline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px 0;
  justify-content: center;
  color: #666666;
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

  .info-value {
    font-size: 14px;
    color: #333333;
    text-align: right;
    word-break: break-word;
  }
}

.usage-control {
//   width: 100%;
//   display: flex;
//   align-items: center;
//   gap: 12px;

  .usage-text {
    font-size: 14px;
    font-weight: 500;
    color: #333333;
    min-width: 50px;
    text-align: right;
  }
}
</style>

