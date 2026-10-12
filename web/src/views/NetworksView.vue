<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Skeleton from 'primevue/skeleton'

interface NetworkSummary {
  id: string
  nwid: string
  name: string
  private: boolean
  memberCount: number
}

const router = useRouter()
const toast = useToast()
const confirm = useConfirm()

const networks = ref<NetworkSummary[]>([])
const loading = ref(true)

const createDialogOpen = ref(false)
const newNetworkName = ref('')
const creating = ref(false)

async function loadNetworks() {
  loading.value = true
  try {
    const { data } = await api.get<NetworkSummary[]>('/networks')
    networks.value = data
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load networks', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}

onMounted(loadNetworks)

async function createNetwork() {
  creating.value = true
  try {
    await api.post('/networks', { name: newNetworkName.value || 'New network' })
    createDialogOpen.value = false
    newNetworkName.value = ''
    await loadNetworks()
    toast.add({ severity: 'success', summary: 'Network created', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not create network', detail: errorMessage(err), life: 5000 })
  } finally {
    creating.value = false
  }
}

function confirmDelete(network: NetworkSummary) {
  confirm.require({
    message: `Delete "${network.name}" (${network.nwid})? This cannot be undone.`,
    header: 'Delete network',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Delete' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      try {
        await api.delete(`/networks/${network.nwid}`)
        await loadNetworks()
        toast.add({ severity: 'success', summary: 'Network deleted', life: 3000 })
      } catch (err) {
        toast.add({ severity: 'error', summary: 'Could not delete network', detail: errorMessage(err), life: 5000 })
      }
    },
  })
}

function openNetwork(network: NetworkSummary) {
  router.push({ name: 'network-detail', params: { id: network.nwid } })
}
</script>

<template>
  <div class="zp-page">
    <div class="zp-page-header">
      <div>
        <h1>Networks</h1>
        <p class="zp-page-subtitle">Networks managed by this ZeroTier controller</p>
      </div>
      <div class="zp-actions">
        <Button label="Add network" icon="pi pi-plus" @click="createDialogOpen = true" />
      </div>
    </div>

    <div class="zp-card" style="padding: 0">
      <div v-if="loading" style="padding: 22px">
        <Skeleton height="2.4rem" class="mb-2" v-for="i in 3" :key="i" style="margin-bottom: 10px" />
      </div>

      <div v-else-if="networks.length === 0" class="zp-empty">
        <i class="pi pi-sitemap" style="font-size: 28px; opacity: 0.4" />
        <p>No networks yet. Create one to get started.</p>
        <Button label="Add network" icon="pi pi-plus" @click="createDialogOpen = true" />
      </div>

      <DataTable v-else :value="networks" @row-click="(e) => openNetwork(e.data)" row-hover style="cursor: pointer">
        <Column field="name" header="Name">
          <template #body="{ data }">
            <span style="font-weight: 550">{{ data.name || '(unnamed)' }}</span>
          </template>
        </Column>
        <Column field="nwid" header="Network ID">
          <template #body="{ data }">
            <span class="zp-mono">{{ data.nwid }}</span>
          </template>
        </Column>
        <Column field="private" header="Access">
          <template #body="{ data }">
            <Tag :severity="data.private ? 'info' : 'warn'" :value="data.private ? 'Private' : 'Public'" />
          </template>
        </Column>
        <Column field="memberCount" header="Members" />
        <Column header="" style="width: 50px">
          <template #body="{ data }">
            <Button
              icon="pi pi-trash"
              severity="danger"
              text
              rounded
              aria-label="Delete network"
              @click.stop="confirmDelete(data)"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <Dialog v-model:visible="createDialogOpen" header="Add network" modal style="width: 420px">
      <div class="zp-field">
        <label for="newNetworkName">Network name</label>
        <InputText id="newNetworkName" v-model="newNetworkName" fluid autofocus placeholder="e.g. cartorios-vpn" />
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 18px">
        <Button label="Cancel" severity="secondary" outlined @click="createDialogOpen = false" />
        <Button label="Create" :loading="creating" @click="createNetwork" />
      </div>
    </Dialog>
  </div>
</template>
