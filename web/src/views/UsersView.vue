<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import { useAuthStore } from '../stores/auth'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Checkbox from 'primevue/checkbox'
import Tag from 'primevue/tag'

interface PanelUser {
  name: string
  mustChangePassword: boolean
}

const toast = useToast()
const confirm = useConfirm()
const auth = useAuthStore()

const users = ref<PanelUser[]>([])
const loading = ref(true)

const createOpen = ref(false)
const newName = ref('')
const newPassword = ref('')
const creating = ref(false)

const resetOpen = ref(false)
const resetTarget = ref<PanelUser | null>(null)
const resetPassword = ref('')
const resetForceChange = ref(true)
const resetting = ref(false)

async function load() {
  loading.value = true
  try {
    const { data } = await api.get<PanelUser[]>('/users')
    users.value = data
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load users', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function createUser() {
  creating.value = true
  try {
    await api.post('/users', { name: newName.value, password: newPassword.value })
    createOpen.value = false
    newName.value = ''
    newPassword.value = ''
    await load()
    toast.add({ severity: 'success', summary: 'User created', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not create user', detail: errorMessage(err), life: 5000 })
  } finally {
    creating.value = false
  }
}

function openReset(user: PanelUser) {
  resetTarget.value = user
  resetPassword.value = ''
  resetForceChange.value = true
  resetOpen.value = true
}

async function submitReset() {
  if (!resetTarget.value) return
  resetting.value = true
  try {
    await api.post(`/users/${resetTarget.value.name}/password`, {
      password: resetPassword.value,
      mustChangePassword: resetForceChange.value,
    })
    resetOpen.value = false
    toast.add({ severity: 'success', summary: 'Password updated', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not set password', detail: errorMessage(err), life: 5000 })
  } finally {
    resetting.value = false
  }
}

function confirmDelete(user: PanelUser) {
  confirm.require({
    message: `Delete user "${user.name}"? This cannot be undone.`,
    header: 'Delete user',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Delete' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      try {
        await api.delete(`/users/${user.name}`)
        await load()
        toast.add({ severity: 'success', summary: 'User deleted', life: 3000 })
      } catch (err) {
        toast.add({ severity: 'error', summary: 'Could not delete user', detail: errorMessage(err), life: 5000 })
      }
    },
  })
}
</script>

<template>
  <div class="zp-page">
    <div class="zp-page-header">
      <div>
        <h1>Users</h1>
        <p class="zp-page-subtitle">Who can sign in to ztpanel</p>
      </div>
      <Button label="Add user" icon="pi pi-plus" @click="createOpen = true" />
    </div>

    <div class="zp-card" style="padding: 0">
      <DataTable :value="users" :loading="loading">
        <Column field="name" header="Username" />
        <Column header="Status">
          <template #body="{ data }">
            <Tag v-if="data.mustChangePassword" severity="warn" value="Must change password" />
            <Tag v-else severity="success" value="Active" />
          </template>
        </Column>
        <Column header="" style="width: 120px">
          <template #body="{ data }">
            <Button icon="pi pi-key" text rounded aria-label="Reset password" @click="openReset(data)" />
            <Button
              icon="pi pi-trash"
              severity="danger"
              text
              rounded
              aria-label="Delete user"
              :disabled="data.name === auth.username"
              @click="confirmDelete(data)"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <Dialog v-model:visible="createOpen" header="Add user" modal style="width: 420px">
      <div class="zp-field">
        <label for="newUserName">Username</label>
        <InputText id="newUserName" v-model="newName" fluid autofocus />
      </div>
      <div class="zp-field">
        <label for="newUserPassword">Initial password</label>
        <Password id="newUserPassword" v-model="newPassword" :feedback="false" toggle-mask fluid />
        <small>They'll be asked to change it on first login.</small>
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Cancel" severity="secondary" outlined @click="createOpen = false" />
        <Button label="Create" :loading="creating" @click="createUser" />
      </div>
    </Dialog>

    <Dialog v-model:visible="resetOpen" header="Set password" modal style="width: 420px">
      <p class="zp-page-subtitle" style="margin-top: 0">For user {{ resetTarget?.name }}</p>
      <div class="zp-field">
        <label for="resetPassword">New password</label>
        <Password id="resetPassword" v-model="resetPassword" :feedback="false" toggle-mask fluid />
      </div>
      <div class="zp-row">
        <Checkbox v-model="resetForceChange" binary input-id="forceChange" />
        <label for="forceChange">Require password change on next login</label>
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 14px">
        <Button label="Cancel" severity="secondary" outlined @click="resetOpen = false" />
        <Button label="Save" :loading="resetting" @click="submitReset" />
      </div>
    </Dialog>
  </div>
</template>
