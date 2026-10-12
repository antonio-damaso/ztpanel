<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import Menu from 'primevue/menu'
import { ref } from 'vue'

const router = useRouter()
const auth = useAuthStore()
const userMenu = ref()

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
</script>

<template>
  <div class="shell">
    <header class="shell-nav">
      <div class="shell-nav-inner">
        <router-link to="/" class="brand">
          <i class="pi pi-sitemap" />
          <span>ztpanel</span>
        </router-link>

        <nav class="shell-links">
          <router-link to="/" class="shell-link">Networks</router-link>
          <router-link to="/users" class="shell-link">Users</router-link>
        </nav>

        <button class="user-button" type="button" @click="toggleUserMenu">
          <i class="pi pi-user" />
          <span>{{ auth.username }}</span>
          <i class="pi pi-chevron-down" style="font-size: 11px" />
        </button>
        <Menu ref="userMenu" :model="userMenuItems" :popup="true" />
      </div>
    </header>

    <main>
      <slot />
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.shell-nav {
  background: #0f172a;
  border-bottom: 1px solid #1e293b;
}

.shell-nav-inner {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 28px;
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

.shell-links {
  display: flex;
  gap: 4px;
  flex: 1;
}

.shell-link {
  color: #94a3b8;
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 500;
  padding: 8px 12px;
  border-radius: 6px;
}
.shell-link:hover {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.06);
}
.shell-link.router-link-exact-active {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.1);
}

.user-button {
  display: flex;
  align-items: center;
  gap: 7px;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #e2e8f0;
  font-size: 13px;
  padding: 6px 10px;
  border-radius: 7px;
  cursor: pointer;
}
.user-button:hover {
  background: rgba(255, 255, 255, 0.06);
}

main {
  flex: 1;
}
</style>
