<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { api, errorMessage } from '../api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'

interface ActivityEntry {
  time: string
  user: string
  action: string
  target: string
  detail?: string
}

const toast = useToast()
const entries = ref<ActivityEntry[]>([])
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const { data } = await api.get<ActivityEntry[]>('/activity?limit=100')
    entries.value = data
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load activity', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}
onMounted(load)

function formatTime(iso: string) {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="zp-page">
    <div class="zp-page-header">
      <div>
        <h1>Activity</h1>
        <p class="zp-page-subtitle">Last 100 changes made in ztpanel</p>
      </div>
      <Button icon="pi pi-refresh" text rounded aria-label="Refresh" @click="load" />
    </div>

    <div class="zp-card" style="padding: 0">
      <div v-if="loading" style="padding: 20px">
        <Skeleton height="1.8rem" v-for="i in 6" :key="i" style="margin-bottom: 8px" />
      </div>
      <div v-else-if="entries.length === 0" class="zp-empty">
        <i class="pi pi-history" style="font-size: 26px; opacity: 0.4" />
        <p>Nothing has happened yet.</p>
      </div>
      <DataTable v-else :value="entries" size="small">
        <Column header="When" style="width: 170px">
          <template #body="{ data }"><span class="zp-mono" style="font-size: 12px">{{ formatTime(data.time) }}</span></template>
        </Column>
        <Column field="user" header="User">
          <template #body="{ data }"><Tag severity="secondary" :value="data.user" /></template>
        </Column>
        <Column field="action" header="Action" />
        <Column field="target" header="Target" />
        <Column field="detail" header="Detail">
          <template #body="{ data }"><span class="zp-mono" style="font-size: 12px">{{ data.detail }}</span></template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>
