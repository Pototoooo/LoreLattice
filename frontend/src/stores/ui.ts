import { defineStore } from 'pinia'

type Theme = 'light' | 'dark'

export const useUIStore = defineStore('ui', {
  state: () => ({
    showSettingsModal: false,
    showKBSettingsModal: false,
    currentKBId: null as string | null,
    settingsInitialSection: null as string | null,
    settingsInitialSubSection: null as string | null,
    theme: 'light' as Theme
  }),

  actions: {
    openSettings(section?: string, subSection?: string) {
      this.settingsInitialSection = section || null
      this.settingsInitialSubSection = subSection || null
      this.showSettingsModal = true
    },

    closeSettings() {
      this.showSettingsModal = false
      this.settingsInitialSection = null
      this.settingsInitialSubSection = null
    },

    toggleSettings() {
      this.showSettingsModal = !this.showSettingsModal
    },

    openKBSettings(kbId: string) {
      this.currentKBId = kbId
      this.showKBSettingsModal = true
    },

    closeKBSettings() {
      this.showKBSettingsModal = false
      this.currentKBId = null
    },

    // 初始化主题（从localStorage加载）
    initTheme() {
      const savedSettings = localStorage.getItem('WeKnora_general_settings')
      if (savedSettings) {
        try {
          const parsed = JSON.parse(savedSettings)
          this.theme = parsed.theme || 'light'
        } catch (e) {
          console.error('加载主题设置失败:', e)
          this.theme = 'light'
        }
      }
      this.applyTheme()
    },

    // 设置主题
    setTheme(theme: Theme) {
      this.theme = theme
      this.applyTheme()
      this.saveThemeToStorage()
    },

    // 切换主题
    toggleTheme() {
      this.theme = this.theme === 'light' ? 'dark' : 'light'
      this.applyTheme()
      this.saveThemeToStorage()
    },

    // 应用主题到DOM
    applyTheme() {
      document.documentElement.setAttribute('theme-mode', this.theme)
    },

    // 保存主题到localStorage
    saveThemeToStorage() {
      const savedSettings = localStorage.getItem('WeKnora_general_settings')
      let settings = { language: 'zh-CN', theme: this.theme }
      
      if (savedSettings) {
        try {
          const parsed = JSON.parse(savedSettings)
          settings = { ...parsed, theme: this.theme }
        } catch (e) {
          console.error('解析现有设置失败:', e)
        }
      }
      
      localStorage.setItem('WeKnora_general_settings', JSON.stringify(settings))
    }
  }
})

