<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Checkbox from 'primevue/checkbox'
import Message from 'primevue/message'
import Tag from 'primevue/tag'

interface ApiToken {
  id: string
  name: string
  last4: string
  readOnly: boolean
  createdAt: string
  expiresAt?: string
  lastUsedAt?: string
}

const toast = useToast()
const confirm = useConfirm()

const tokens = ref<ApiToken[]>([])
const loading = ref(true)

const createOpen = ref(false)
const newName = ref('')
const newReadOnly = ref(false)
const newExpiresInDays = ref<number | null>(null)
const creating = ref(false)

const createdOpen = ref(false)
const createdToken = ref('')

async function load() {
  loading.value = true
  try {
    const { data } = await api.get<ApiToken[]>('/tokens')
    tokens.value = data
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load tokens', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function createToken() {
  if (!newName.value) return
  creating.value = true
  try {
    const { data } = await api.post('/tokens', {
      name: newName.value,
      readOnly: newReadOnly.value,
      expiresInDays: newExpiresInDays.value ?? 0,
    })
    createOpen.value = false
    newName.value = ''
    newReadOnly.value = false
    newExpiresInDays.value = null
    createdToken.value = data.token
    createdOpen.value = true
    await load()
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not create token', detail: errorMessage(err), life: 5000 })
  } finally {
    creating.value = false
  }
}

async function copyToken() {
  try {
    await navigator.clipboard.writeText(createdToken.value)
    toast.add({ severity: 'success', summary: 'Copied', life: 2000 })
  } catch {
    // Clipboard access can fail silently (permissions, insecure
    // context, ...); the token stays visible to copy by hand.
  }
}

function confirmRevoke(token: ApiToken) {
  confirm.require({
    message: `Revoke "${token.name}"? Anything using this token will stop working immediately.`,
    header: 'Revoke token',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Revoke' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      try {
        await api.delete(`/tokens/${token.id}`)
        await load()
        toast.add({ severity: 'success', summary: 'Token revoked', life: 3000 })
      } catch (err) {
        toast.add({ severity: 'error', summary: 'Could not revoke token', detail: errorMessage(err), life: 5000 })
      }
    },
  })
}

function formatDate(iso?: string) {
  return iso ? new Date(iso).toLocaleString() : 'Never'
}
</script>

<template>
  <div class="zp-card" style="padding: 0">
    <div class="zp-row-between" style="padding: 18px 22px 0">
      <p class="zp-card-title" style="margin-bottom: 4px">API tokens</p>
      <Button label="New token" icon="pi pi-plus" size="small" @click="createOpen = true" />
    </div>
    <p class="zp-page-subtitle" style="padding: 0 22px 14px; margin: 0">
      Use a token to call the REST API or the
      <code class="zp-mono">/mcp</code> endpoint (e.g. from Claude or another MCP client) without a browser session.
    </p>

    <div v-if="!loading && tokens.length === 0" class="zp-empty">
      <i class="pi pi-key" style="font-size: 26px; opacity: 0.4" />
      <p>No API tokens yet.</p>
    </div>

    <DataTable v-else :value="tokens" :loading="loading" data-key="id" style="margin-top: 8px">
      <Column field="name" header="Name" />
      <Column header="Token">
        <template #body="{ data }"><span class="zp-mono">ztpanel_...{{ data.last4 }}</span></template>
      </Column>
      <Column header="Scope">
        <template #body="{ data }">
          <Tag :value="data.readOnly ? 'Read-only' : 'Read/write'" :severity="data.readOnly ? 'info' : 'secondary'" />
        </template>
      </Column>
      <Column header="Expires">
        <template #body="{ data }">{{ data.expiresAt ? formatDate(data.expiresAt) : 'Never' }}</template>
      </Column>
      <Column header="Created">
        <template #body="{ data }">{{ formatDate(data.createdAt) }}</template>
      </Column>
      <Column header="Last used">
        <template #body="{ data }">{{ formatDate(data.lastUsedAt) }}</template>
      </Column>
      <Column header="" style="width: 50px">
        <template #body="{ data }">
          <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Revoke token" @click="confirmRevoke(data)" />
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="createOpen" header="New API token" modal style="width: 420px">
      <div class="zp-field">
        <label for="newTokenName">Name</label>
        <InputText id="newTokenName" v-model="newName" fluid autofocus placeholder="e.g. Claude Desktop" @keyup.enter="createToken" />
      </div>
      <div class="zp-field">
        <label for="newTokenExpiry">Expires in (days)</label>
        <InputNumber id="newTokenExpiry" v-model="newExpiresInDays" fluid :min="0" placeholder="Never" />
      </div>
      <div class="zp-row" style="gap: 8px; margin-top: 4px">
        <Checkbox v-model="newReadOnly" input-id="newTokenReadOnly" binary />
        <label for="newTokenReadOnly" style="cursor: pointer">
          Read-only (can list/get, but not create, update or delete anything)
        </label>
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Cancel" severity="secondary" outlined @click="createOpen = false" />
        <Button label="Create" :loading="creating" @click="createToken" />
      </div>
    </Dialog>

    <Dialog v-model:visible="createdOpen" header="Token created" modal style="width: min(520px, 92vw)" :closable="false">
      <Message severity="warn" :closable="false" style="margin-bottom: 14px">
        Copy this token now -- you won't be able to see it again.
      </Message>
      <div class="zp-field">
        <label>Token</label>
        <div class="zp-row">
          <InputText :model-value="createdToken" readonly fluid class="zp-mono" />
          <Button icon="pi pi-copy" aria-label="Copy token" @click="copyToken" />
        </div>
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Done" @click="createdOpen = false" />
      </div>
    </Dialog>
  </div>
</template>
