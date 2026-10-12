<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import AppShell from './components/AppShell.vue'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'

const route = useRoute()
const auth = useAuthStore()
</script>

<template>
  <Toast position="top-right" />
  <ConfirmDialog />

  <template v-if="route.meta.public || !auth.isAuthenticated">
    <router-view />
  </template>
  <AppShell v-else>
    <router-view />
  </AppShell>
</template>
