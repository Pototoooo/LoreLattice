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

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getSystemInfo, type SystemInfo } from '@/api/system'

// 响应式数据
const systemInfo = ref<SystemInfo | null>(null)
const loading = ref(true)
const error = ref('')

// 方法
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    
    const systemResponse = await getSystemInfo()
    
    if (systemResponse.data) {
      systemInfo.value = systemResponse.data
    } else {
      error.value = '获取系统信息失败'
    }
  } catch (err: any) {
    error.value = err.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
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

    .commit-info {
      color: #999999;
      font-size: 12px;
      margin-left: 6px;
    }
  }
}
</style>
