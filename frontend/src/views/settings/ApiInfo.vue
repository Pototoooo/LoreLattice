<template>
  <div class="api-info">
    <div class="section-header">
      <h2>API 信息</h2>
      <p class="section-description">查看和管理您的 API 密钥</p>
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
            <t-button 
              size="small" 
              variant="text"
              @click="copyApiKey"
              title="复制 API Key"
            >
              <t-icon name="file-copy" />
            </t-button>
          </div>
        </div>
      </div>

      <!-- API 文档 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>API 文档</label>
          <p class="desc">查看完整的 API 调用文档和示例</p>
        </div>
        <div class="setting-control">
          <t-button 
            size="small" 
            theme="default" 
            variant="outline"
            @click="openApiDoc"
          >
            查看文档
          </t-button>
        </div>
      </div>

      <!-- 用户信息 -->
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

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getCurrentUser, type TenantInfo, type UserInfo } from '@/api/auth'
import { MessagePlugin } from 'tdesign-vue-next'

// 响应式数据
const tenantInfo = ref<TenantInfo | null>(null)
const userInfo = ref<UserInfo | null>(null)
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
    
    const userResponse = await getCurrentUser()
    
    if (userResponse.success && userResponse.data) {
      userInfo.value = userResponse.data.user
      tenantInfo.value = userResponse.data.tenant
    } else {
      error.value = userResponse.message || '获取用户信息失败'
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

const copyApiKey = async () => {
  if (!displayApiKey.value) {
    MessagePlugin.warning('暂无 API Key')
    return
  }
  
  try {
    await navigator.clipboard.writeText(displayApiKey.value)
    MessagePlugin.success('API Key 已复制到剪贴板')
  } catch (err) {
    MessagePlugin.error('复制失败，请手动复制')
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

// 生命周期
onMounted(() => {
  loadInfo()
})
</script>

<style lang="less" scoped>
.api-info {
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

.api-key-control {
  width: 100%;
  display: flex;
  gap: 8px;
  align-items: center;
}

.info-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #333333;
  margin-top: 24px;
  margin-bottom: 12px;

  &:first-child {
    margin-top: 0;
  }
}
</style>

