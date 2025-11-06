import { defineStore } from 'pinia'

export const useUIStore = defineStore('ui', {
  state: () => ({
    showSettingsModal: false,
    showKBEditorModal: false,
    kbEditorMode: 'create' as 'create' | 'edit',
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
      this.kbEditorMode = 'edit'
      this.showKBEditorModal = true
    },

    openCreateKB() {
      this.currentKBId = null
      this.kbEditorMode = 'create'
      this.showKBEditorModal = true
    },

    closeKBEditor() {
      this.showKBEditorModal = false
      this.currentKBId = null
    }
  }
})

