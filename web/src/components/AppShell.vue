<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useOrgsStore } from '../stores/orgs'
import Menu from 'primevue/menu'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const orgs = useOrgsStore()
const userMenu = ref()
const sidebarOpen = ref(false)

onMounted(() => orgs.load())

// Close the mobile drawer whenever navigation happens, so tapping a
// nav link doesn't leave it hanging open over the new page.
watch(
  () => route.fullPath,
  () => {
    sidebarOpen.value = false
  },
)

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
    <header class="mobile-bar">
      <button class="icon-btn" type="button" aria-label="Open menu" @click="sidebarOpen = true">
        <i class="pi pi-bars" />
      </button>
      <router-link to="/" class="brand-mobile">
        <i class="pi pi-sitemap" /><span>ztpanel</span>
      </router-link>
    </header>

    <div v-if="sidebarOpen" class="backdrop" @click="sidebarOpen = false" />

    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-top">
        <router-link to="/" class="brand">
          <i class="pi pi-sitemap" />
          <span>ztpanel</span>
        </router-link>
        <button class="icon-btn close-btn" type="button" aria-label="Close menu" @click="sidebarOpen = false">
          <i class="pi pi-times" />
        </button>
      </div>

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
        <router-link to="/activity" class="nav-link">
          <i class="pi pi-history" /><span>Activity</span>
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

    <Dialog v-model:visible="createOrgOpen" header="New organization" modal :style="{ width: 'min(420px, 92vw)' }">
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
/* Mobile-first: the sidebar is an off-canvas drawer by default, and
   only becomes a static column at the md breakpoint (768px). */

.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.mobile-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #0f172a;
  padding: 0 6px;
  height: 52px;
  position: sticky;
  top: 0;
  z-index: 30;
}

.icon-btn {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: #e2e8f0;
  font-size: 18px;
  cursor: pointer;
  border-radius: 8px;
}
.icon-btn:hover {
  background: rgba(255, 255, 255, 0.08);
}

.brand-mobile {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #ffffff;
  font-weight: 650;
  font-size: 15px;
  text-decoration: none;
}
.brand-mobile i {
  color: var(--p-primary-400, #60a5fa);
}

.backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  z-index: 35;
}

.sidebar {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  width: 84vw;
  max-width: 280px;
  background: #0f172a;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
  padding: 10px 12px 16px;
  z-index: 40;
  overflow-y: auto;
  transform: translateX(-100%);
  transition: transform 0.22s ease;
}
.sidebar.open {
  transform: translateX(0);
}

.sidebar-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 4px 18px 8px;
}

.close-btn {
  display: flex;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #ffffff;
  font-weight: 650;
  font-size: 15px;
  text-decoration: none;
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
  width: 32px;
  height: 32px;
  border-radius: 7px;
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
  gap: 10px;
  padding: 11px 8px;
  border-radius: 7px;
  color: #cbd5e1;
  text-decoration: none;
  font-size: 14.5px;
  font-weight: 500;
  min-height: 44px;
}
.nav-link i {
  font-size: 14px;
  width: 16px;
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
  font-size: 13.5px;
  padding: 10px 8px;
  border-radius: 7px;
  cursor: pointer;
  min-height: 44px;
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

.icon-btn:focus-visible,
.nav-link:focus-visible,
.nav-add:focus-visible,
.user-button:focus-visible {
  outline: 2px solid var(--p-primary-400, #60a5fa);
  outline-offset: 2px;
}

@media (min-width: 768px) {
  .mobile-bar,
  .backdrop,
  .close-btn {
    display: none;
  }

  .shell {
    flex-direction: row;
  }

  .sidebar {
    position: static;
    width: 236px;
    flex-shrink: 0;
    transform: none;
    transition: none;
  }

  .sidebar-top {
    padding: 6px 8px 18px;
  }
}
</style>
