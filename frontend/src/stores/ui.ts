import { defineStore } from 'pinia'

export const useUIStore = defineStore('ui', {
  state: () => ({
    showSettingsModal: false,
    showKBSettingsModal: false,
    currentKBId: null as string | null,
    settingsInitialSection: null as string | null,
    settingsInitialSubSection: null as string | null
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
    }
  }
})

