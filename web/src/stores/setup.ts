import { defineStore } from 'pinia'
import { api } from '../api/client'

interface SetupStatusResponse {
  needsSetup: boolean
  controllerReachable?: boolean
  controllerVersion?: string
  controllerError?: string
}

export const useSetupStore = defineStore('setup', {
  state: () => ({
    needed: false,
    checked: false,
    controllerReachable: null as boolean | null,
    controllerVersion: '',
    controllerError: '',
  }),
  actions: {
    async check() {
      try {
        const { data } = await api.get<SetupStatusResponse>('/setup/status')
        this.needed = data.needsSetup
        this.controllerReachable = data.controllerReachable ?? null
        this.controllerVersion = data.controllerVersion ?? ''
        this.controllerError = data.controllerError ?? ''
      } catch {
        // If the check itself fails, don't block the whole app behind
        // the setup page -- fall through to the normal login flow.
        this.needed = false
      } finally {
        this.checked = true
      }
    },
    markDone() {
      this.needed = false
    },
  },
})
