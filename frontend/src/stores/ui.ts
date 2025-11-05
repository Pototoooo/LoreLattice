import { defineStore } from 'pinia'

export const useUIStore = defineStore('ui', {
  state: () => ({
    showSettingsModal: false
  }),

  actions: {
    openSettings() {
      this.showSettingsModal = true
    },

    closeSettings() {
      this.showSettingsModal = false
    },

    toggleSettings() {
      this.showSettingsModal = !this.showSettingsModal
    }
  }
})

