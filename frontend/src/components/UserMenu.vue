<template>
  <div class="user-menu" ref="menuRef">
    <!-- 用户按钮 -->
    <div class="user-button" @click="toggleMenu">
      <div class="user-avatar">
        <img v-if="userAvatar" :src="userAvatar" alt="用户头像" />
        <span v-else class="avatar-placeholder">{{ userInitial }}</span>
      </div>
      <div class="user-info">
        <div class="user-name">{{ userName }}</div>
        <div class="user-email">{{ userEmail }}</div>
      </div>
      <t-icon :name="menuVisible ? 'chevron-up' : 'chevron-down'" class="dropdown-icon" />
    </div>

    <!-- 下拉菜单 -->
    <Transition name="dropdown">
      <div v-if="menuVisible" class="user-dropdown" @click.stop>
        <div class="menu-item" @click="handleQuickNav('models')">
          <t-icon name="control-platform" class="menu-icon" />
          <span>模型配置</span>
        </div>
        <div class="menu-item" @click="handleQuickNav('ollama')">
          <t-icon name="server" class="menu-icon" />
          <span>Ollama</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="handleSettings">
          <t-icon name="setting" class="menu-icon" />
          <span>全部设置</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="openApiDoc">
          <t-icon name="book" class="menu-icon" />
          <span>API 文档</span>
        </div>
        <div class="menu-item" @click="openWebsite">
          <t-icon name="home" class="menu-icon" />
          <span>官网</span>
        </div>
        <div class="menu-item" @click="openGithub">
          <t-icon name="logo-github" class="menu-icon" />
          <span>GitHub</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item danger" @click="handleLogout">
          <t-icon name="logout" class="menu-icon" />
          <span>注销</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import { getCurrentUser, logout as logoutApi } from '@/api/auth'

const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()

const menuRef = ref<HTMLElement>()
const menuVisible = ref(false)

// 用户信息
const userInfo = ref({
  username: '用户',
  email: 'user@example.com',
  avatar: ''
})

const userName = computed(() => userInfo.value.username)
const userEmail = computed(() => userInfo.value.email)
const userAvatar = computed(() => userInfo.value.avatar)

// 用户名首字母（用于无头像时显示）
const userInitial = computed(() => {
  return userName.value.charAt(0).toUpperCase()
})

// 切换菜单显示
const toggleMenu = () => {
  menuVisible.value = !menuVisible.value
}

// 快捷导航到设置的特定部分
const handleQuickNav = (section: string) => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push('/platform/settings')
  
  // 延迟一下，确保设置页面已经渲染
  setTimeout(() => {
    // 触发设置页面切换到对应section
    const event = new CustomEvent('settings-nav', { detail: { section } })
    window.dispatchEvent(event)
  }, 100)
}

// 打开设置
const handleSettings = () => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push('/platform/settings')
}

// 打开 API 文档
const openApiDoc = () => {
  menuVisible.value = false
  window.open('https://github.com/Tencent/WeKnora/blob/main/docs/API.md', '_blank')
}

// 打开官网
const openWebsite = () => {
  menuVisible.value = false
  window.open('https://weknora.weixin.qq.com/', '_blank')
}

// 打开 GitHub
const openGithub = () => {
  menuVisible.value = false
  window.open('https://github.com/Tencent/WeKnora', '_blank')
}

// 注销
const handleLogout = async () => {
  menuVisible.value = false
  
  try {
    // 调用后端API注销
    await logoutApi()
  } catch (error) {
    // 即使API调用失败，也继续执行本地清理
    console.error('注销API调用失败:', error)
  }
  
  // 清理所有状态和本地存储
  authStore.logout()
  
  MessagePlugin.success('已退出登录')
  
  // 跳转到登录页
  router.push('/login')
}

// 加载用户信息
const loadUserInfo = async () => {
  try {
    const response = await getCurrentUser()
    if (response.success && response.data && response.data.user) {
      userInfo.value = {
        username: response.data.user.username || '用户',
        email: response.data.user.email || 'user@example.com',
        avatar: response.data.user.avatar || ''
      }
    }
  } catch (error) {
    console.error('Failed to load user info:', error)
  }
}

// 点击外部关闭菜单
const handleClickOutside = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    menuVisible.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  loadUserInfo()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style lang="less" scoped>
.user-menu {
  position: relative;
  width: 100%;
}

.user-button {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background: transparent;

  &:hover {
    background: #f5f7fa;
  }

  &:active {
    transform: scale(0.98);
  }
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: linear-gradient(135deg, #07C05F 0%, #05A34E 100%);
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .avatar-placeholder {
    color: #ffffff;
    font-size: 16px;
    font-weight: 600;
  }
}

.user-info {
  flex: 1;
  min-width: 0;
  text-align: left;

  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: #333333;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-email {
    font-size: 12px;
    color: #666666;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.dropdown-icon {
  font-size: 16px;
  color: #666666;
  flex-shrink: 0;
  transition: transform 0.2s;
}

.user-dropdown {
  position: absolute;
  bottom: 100%;
  left: 8px;
  right: 8px;
  margin-bottom: 8px;
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  border: 1px solid #e5e7eb;
  overflow: hidden;
  z-index: 1000;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
  color: #333333;

  &:hover {
    background: #f5f7fa;
  }

  &.danger {
    color: #e34d59;

    &:hover {
      background: #fef0f0;
    }

    .menu-icon {
      color: #e34d59;
    }
  }

  .menu-icon {
    font-size: 16px;
    color: #666666;
  }
}

.menu-divider {
  height: 1px;
  background: #e5e7eb;
  margin: 4px 0;
}

// 下拉动画
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

.dropdown-enter-to,
.dropdown-leave-from {
  opacity: 1;
  transform: translateY(0);
}
</style>

