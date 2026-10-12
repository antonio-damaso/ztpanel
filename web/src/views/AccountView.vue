<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { api, errorMessage } from '../api/client'
import { useAuthStore } from '../stores/auth'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const auth = useAuthStore()

const forced = route.query.forced === '1'
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref('')
const saving = ref(false)

async function submit() {
  error.value = ''
  if (newPassword.value.length < 8) {
    error.value = 'New password must be at least 8 characters'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  saving.value = true
  try {
    await api.post('/me/password', { currentPassword: currentPassword.value, newPassword: newPassword.value })
    auth.clearMustChangePassword()
    toast.add({ severity: 'success', summary: 'Password updated', life: 3000 })
    if (forced) router.push({ name: 'networks' })
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (err) {
    error.value = errorMessage(err, 'Could not update password')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="zp-page" style="max-width: 480px">
    <div class="zp-page-header">
      <div>
        <h1>Account</h1>
        <p class="zp-page-subtitle">Signed in as {{ auth.username }}</p>
      </div>
    </div>

    <div class="zp-card">
      <p class="zp-card-title">Change password</p>

      <Message v-if="forced" severity="info" :closable="false" style="margin-bottom: 16px">
        Set a new password to continue.
      </Message>
      <Message v-if="error" severity="error" :closable="false" style="margin-bottom: 16px">{{ error }}</Message>

      <div class="zp-field">
        <label for="currentPassword">Current password</label>
        <Password id="currentPassword" v-model="currentPassword" :feedback="false" toggle-mask fluid autocomplete="current-password" />
      </div>
      <div class="zp-field">
        <label for="newPassword">New password</label>
        <Password id="newPassword" v-model="newPassword" toggle-mask fluid autocomplete="new-password" />
      </div>
      <div class="zp-field">
        <label for="confirmPassword">Confirm new password</label>
        <Password id="confirmPassword" v-model="confirmPassword" :feedback="false" toggle-mask fluid autocomplete="new-password" />
      </div>

      <Button label="Update password" :loading="saving" @click="submit" />
    </div>
  </div>
</template>
