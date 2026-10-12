import { defineStore } from 'pinia'
import { api, errorMessage } from '../api/client'

interface MeResponse {
  username: string
}

interface LoginResponse {
  username: string
  mustChangePassword: boolean
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    username: null as string | null,
    mustChangePassword: false,
    // Whether we've checked for an existing session yet. Avoids a
    // flash of the login screen while that check is in flight.
    checked: false,
  }),
  getters: {
    isAuthenticated: (state) => state.username !== null,
  },
  actions: {
    async login(username: string, password: string) {
      try {
        const { data } = await api.post<LoginResponse>('/login', { username, password })
        this.username = data.username
        this.mustChangePassword = data.mustChangePassword
        return { ok: true as const }
      } catch (err) {
        return { ok: false as const, message: errorMessage(err, 'Could not log in') }
      }
    },
    async logout() {
      try {
        await api.post('/logout')
      } finally {
        this.username = null
      }
    },
    async checkSession() {
      try {
        const { data } = await api.get<MeResponse>('/me')
        this.username = data.username
      } catch {
        this.username = null
      } finally {
        this.checked = true
      }
    },
    clearMustChangePassword() {
      this.mustChangePassword = false
    },
  },
})
