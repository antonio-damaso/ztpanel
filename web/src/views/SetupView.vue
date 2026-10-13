<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSetupStore } from '../stores/setup'
import { useAuthStore } from '../stores/auth'
import { api, errorMessage } from '../api/client'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'

const router = useRouter()
const setup = useSetupStore()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const saving = ref(false)
const checkingController = ref(true)

onMounted(async () => {
  await setup.check()
  checkingController.value = false
})

async function handleSubmit() {
  error.value = ''
  if (!username.value) {
    error.value = 'Choose a username'
    return
  }
  if (password.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  saving.value = true
  try {
    const { data } = await api.post('/setup', { username: username.value, password: password.value })
    auth.username = data.username
    auth.mustChangePassword = false
    auth.checked = true
    setup.markDone()
    router.push({ name: 'networks' })
  } catch (err) {
    error.value = errorMessage(err, 'Could not finish setup')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="auth-wrap">
    <form class="auth-card" @submit.prevent="handleSubmit">
      <div class="auth-brand">
        <i class="pi pi-sitemap" />
        <span>ztpanel</span>
      </div>
      <p class="auth-subtitle">Welcome -- let's set up your admin account</p>

      <div v-if="checkingController" class="zp-row" style="justify-content: center; margin-bottom: 16px">
        <ProgressSpinner style="width: 24px; height: 24px" :stroke-width="6" />
      </div>
      <Message v-else-if="setup.controllerReachable" severity="success" :closable="false" class="auth-controller-status">
        Connected to the ZeroTier controller{{ setup.controllerVersion ? ` (v${setup.controllerVersion})` : '' }}.
      </Message>
      <Message v-else severity="warn" :closable="false" class="auth-controller-status">
        Could not reach the ZeroTier controller{{ setup.controllerError ? `: ${setup.controllerError}` : '' }}.
        You can still finish setup and fix the connection afterwards.
      </Message>

      <Message v-if="error" severity="error" :closable="false" class="auth-error">{{ error }}</Message>

      <div class="zp-field">
        <label for="setupUsername">Admin username</label>
        <InputText id="setupUsername" v-model="username" autofocus fluid autocomplete="username" />
      </div>

      <div class="zp-field">
        <label for="setupPassword">Admin password</label>
        <Password id="setupPassword" v-model="password" toggle-mask fluid autocomplete="new-password" />
      </div>

      <div class="zp-field">
        <label for="setupConfirmPassword">Confirm password</label>
        <Password id="setupConfirmPassword" v-model="confirmPassword" :feedback="false" toggle-mask fluid autocomplete="new-password" />
      </div>

      <Button type="submit" label="Finish setup" :loading="saving" fluid />
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
  max-width: 380px;
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

.auth-controller-status,
.auth-error {
  margin-bottom: 16px;
}
</style>
