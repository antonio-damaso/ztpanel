import { defineStore } from 'pinia'
import { api } from '../api/client'

export interface OrgMember {
  username: string
  role: 'ADMIN' | 'USER'
}

export interface Organization {
  id: string
  name: string
  createdAt: string
  members: OrgMember[]
  networkIds: string[]
}

export const useOrgsStore = defineStore('orgs', {
  state: () => ({
    organizations: [] as Organization[],
    loaded: false,
  }),
  actions: {
    async load(force = false) {
      if (this.loaded && !force) return
      const { data } = await api.get<Organization[]>('/organizations')
      this.organizations = data
      this.loaded = true
    },
    async create(name: string) {
      const { data } = await api.post<Organization>('/organizations', { name })
      this.organizations.push(data)
      return data
    },
  },
})
