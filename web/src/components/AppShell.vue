<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useOrgsStore } from '../stores/orgs'
import Menu from 'primevue/menu'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'

const router = useRouter()
const auth = useAuthStore()
const orgs = useOrgsStore()
const userMenu = ref()

onMounted(() => orgs.load())

const userMenuItems = [
  { label: 'Account', icon: 'pi pi-user', command: () => router.push({ name: 'account' }) },
  { label: 'Log out', icon: 'pi pi-sign-out', command: () => handleLogout() },
]

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}

function toggleUserMenu(event: Event) {
  userMenu.value.toggle(event)
}

const createOrgOpen = ref(false)
const newOrgName = ref('')
const creatingOrg = ref(false)

async function createOrganization() {
  if (!newOrgName.value) return
  creatingOrg.value = true
  try {
    const org = await orgs.create(newOrgName.value)
    createOrgOpen.value = false
    newOrgName.value = ''
    router.push({ name: 'organization', params: { id: org.id } })
  } finally {
    creatingOrg.value = false
  }
}
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <router-link to="/" class="brand">
        <i class="pi pi-sitemap" />
        <span>ztpanel</span>
      </router-link>

      <nav class="nav-section">
        <p class="nav-label">General</p>
        <router-link to="/" class="nav-link">
          <i class="pi pi-home" /><span>My networks</span>
        </router-link>
      </nav>

      <nav class="nav-section">
        <div class="nav-label-row">
          <p class="nav-label">Organizations</p>
          <button class="nav-add" type="button" aria-label="New organization" @click="createOrgOpen = true">
            <i class="pi pi-plus" />
          </button>
        </div>
        <router-link
          v-for="org in orgs.organizations"
          :key="org.id"
          :to="{ name: 'organization', params: { id: org.id } }"
          class="nav-link"
        >
          <i class="pi pi-building" /><span>{{ org.name }}</span>
        </router-link>
        <p v-if="orgs.loaded && orgs.organizations.length === 0" class="nav-empty">No organizations yet</p>
      </nav>

      <nav class="nav-section">
        <p class="nav-label">Admin</p>
        <router-link to="/users" class="nav-link">
          <i class="pi pi-users" /><span>Panel users</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <button class="user-button" type="button" @click="toggleUserMenu">
          <i class="pi pi-user" />
          <span>{{ auth.username }}</span>
          <i class="pi pi-chevron-up" style="font-size: 11px" />
        </button>
        <Menu ref="userMenu" :model="userMenuItems" :popup="true" />
      </div>
    </aside>

    <main class="content">
      <slot />
    </main>

    <Dialog v-model:visible="createOrgOpen" header="New organization" modal style="width: 420px">
      <div class="zp-field">
        <label for="newOrgName">Name</label>
        <InputText id="newOrgName" v-model="newOrgName" fluid autofocus @keyup.enter="createOrganization" />
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Cancel" severity="secondary" outlined @click="createOrgOpen = false" />
        <Button label="Create" :loading="creatingOrg" @click="createOrganization" />
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
}

.sidebar {
  width: 236px;
  flex-shrink: 0;
  background: #0f172a;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
  padding: 16px 12px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #ffffff;
  font-weight: 650;
  font-size: 15px;
  text-decoration: none;
  padding: 6px 8px 18px;
}
.brand i {
  color: var(--p-primary-400, #60a5fa);
  font-size: 16px;
}

.nav-section {
  margin-bottom: 18px;
}

.nav-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
}

.nav-label {
  font-size: 11px;
  font-weight: 650;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #64748b;
  margin: 0 0 6px;
  padding: 0 8px;
}

.nav-label-row .nav-label {
  padding: 0;
}

.nav-add {
  background: transparent;
  border: none;
  color: #64748b;
  cursor: pointer;
  width: 22px;
  height: 22px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.nav-add:hover {
  color: #e2e8f0;
  background: rgba(255, 255, 255, 0.08);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 7px 8px;
  border-radius: 7px;
  color: #cbd5e1;
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 500;
}
.nav-link i {
  font-size: 13px;
  width: 15px;
  text-align: center;
}
.nav-link:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #ffffff;
}
.nav-link.router-link-exact-active {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}

.nav-empty {
  padding: 0 8px;
  font-size: 12px;
  color: #475569;
  margin: 0;
}

.sidebar-footer {
  margin-top: auto;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  padding-top: 12px;
}

.user-button {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: none;
  color: #e2e8f0;
  font-size: 13px;
  padding: 7px 8px;
  border-radius: 7px;
  cursor: pointer;
}
.user-button span {
  flex: 1;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-button:hover {
  background: rgba(255, 255, 255, 0.06);
}

.content {
  flex: 1;
  min-width: 0;
}
</style>
