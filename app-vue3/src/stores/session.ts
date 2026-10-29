import { defineStore } from 'pinia'
import { api } from '@/api/http'
import { broadcastLogout } from '@/utils/authSync'

export interface Me {
  user: { id: string; username: string; name: string; role: string }
  permissions: string[]
  outlet: { name: string }
}

export const useSessionStore = defineStore('session', {
  state: () => ({
    ready: false,
    user: null as Me['user'] | null,
    permissions: [] as string[],
    outletName: '',
    error: '',
  }),
  getters: {
    can: (s) => (perm: string) => s.permissions.includes(perm),
  },
  actions: {
    /** Bootstrap from the server once per page load: cookie -> GET /api/me. */
    async init() {
      if (this.ready) return
      try {
        const me = await api.get<Me>('/api/me')
        this.user = me.user
        this.permissions = me.permissions
        this.outletName = me.outlet.name
        this.ready = true
      } catch (e) {
        this.error = e instanceof Error ? e.message : 'Could not load your session'
        throw e
      }
    },
    async logout() {
      try {
        await api.post('/api/auth/logout')
      } catch {
        /* already signed out */
      }
      broadcastLogout()
      window.location.assign('/login')
    },
  },
})
