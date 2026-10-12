<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  loading.value = true
  const result = await auth.login(username.value, password.value)
  loading.value = false

  if (!result.ok) {
    error.value = result.message
    return
  }

  if (auth.mustChangePassword) {
    router.push({ name: 'account', query: { forced: '1' } })
    return
  }

  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
}
</script>

<template>
  <div class="auth-wrap">
    <form class="auth-card" @submit.prevent="handleSubmit">
      <div class="auth-brand">
        <i class="pi pi-sitemap" />
        <span>ztpanel</span>
      </div>
      <p class="auth-subtitle">Sign in to manage your ZeroTier network controller</p>

      <Message v-if="error" severity="error" :closable="false" class="auth-error">{{ error }}</Message>

      <div class="zp-field">
        <label for="username">Username</label>
        <InputText id="username" v-model="username" autofocus fluid autocomplete="username" />
      </div>

      <div class="zp-field">
        <label for="password">Password</label>
        <Password id="password" v-model="password" :feedback="false" toggle-mask fluid autocomplete="current-password" />
      </div>

      <Button type="submit" label="Sign in" :loading="loading" fluid />
    </form>
  </div>
</template>

<style scoped>
.auth-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: var(--zp-bg);
}

.auth-card {
  width: 100%;
  max-width: 360px;
  background: var(--zp-surface);
  border: 1px solid var(--zp-border);
  border-radius: 12px;
  box-shadow: var(--zp-shadow);
  padding: 34px 30px;
}

.auth-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-weight: 650;
  font-size: 17px;
  margin-bottom: 4px;
}
.auth-brand i {
  color: #2563eb;
}

.auth-subtitle {
  text-align: center;
  color: var(--zp-text-muted);
  font-size: 13px;
  margin: 0 0 22px;
}

.auth-error {
  margin-bottom: 16px;
}
</style>
