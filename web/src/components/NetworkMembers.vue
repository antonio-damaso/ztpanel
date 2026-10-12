<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Skeleton from 'primevue/skeleton'
import StatusPill from './StatusPill.vue'

const props = defineProps<{ networkId: string; controllerAddress?: string }>()

interface Member {
  id: string
  address: string
  name: string
  authorized: boolean
  activeBridge: boolean
  ipAssignments: string[]
  peer?: { latency: number; versionMajor: number; version: string } | null
}

const toast = useToast()
const confirm = useConfirm()
const members = ref<Member[]>([])
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const { data } = await api.get<Member[]>(`/networks/${props.networkId}/members`)
    members.value = data
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load members', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}

onMounted(load)
defineExpose({ reload: load })

async function patchMember(member: Member, fields: Record<string, unknown>) {
  try {
    await api.patch(`/networks/${props.networkId}/members/${member.address}`, fields)
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not update member', detail: errorMessage(err), life: 5000 })
    await load()
  }
}

function onToggleAuthorized(member: Member) {
  patchMember(member, { authorized: member.authorized })
}

function onToggleBridge(member: Member) {
  patchMember(member, { activeBridge: member.activeBridge })
}

let nameTimers: Record<string, ReturnType<typeof setTimeout>> = {}
function onNameInput(member: Member) {
  clearTimeout(nameTimers[member.address])
  nameTimers[member.address] = setTimeout(() => patchMember(member, { name: member.name }), 500)
}

function confirmDelete(member: Member) {
  confirm.require({
    message: `Remove "${member.name || member.address}" from this network?`,
    header: 'Remove member',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Remove' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      try {
        await api.delete(`/networks/${props.networkId}/members/${member.address}`)
        await load()
        toast.add({ severity: 'success', summary: 'Member removed', life: 3000 })
      } catch (err) {
        toast.add({ severity: 'error', summary: 'Could not remove member', detail: errorMessage(err), life: 5000 })
      }
    },
  })
}
</script>

<template>
  <div class="zp-card" style="padding: 0">
    <div style="padding: 18px 22px 0">
      <p class="zp-card-title" style="margin-bottom: 4px">Members ({{ members.length }})</p>
    </div>

    <div v-if="loading" style="padding: 18px 22px">
      <Skeleton height="2.2rem" v-for="i in 3" :key="i" style="margin-bottom: 10px" />
    </div>

    <div v-else-if="members.length === 0" class="zp-empty">
      <i class="pi pi-users" style="font-size: 26px; opacity: 0.4" />
      <p>No members yet. Join this network from a device to see it here.</p>
    </div>

    <DataTable v-else :value="members" data-key="address" style="margin-top: 8px">
      <Column field="name" header="Name">
        <template #body="{ data }">
          <InputText v-model="data.name" size="small" placeholder="(unnamed)" @input="onNameInput(data)" />
        </template>
      </Column>
      <Column field="address" header="Address">
        <template #body="{ data }"><span class="zp-mono">{{ data.address }}</span></template>
      </Column>
      <Column header="Authorized" style="width: 100px">
        <template #body="{ data }">
          <Checkbox v-model="data.authorized" binary @change="onToggleAuthorized(data)" />
        </template>
      </Column>
      <Column header="Bridge" style="width: 90px">
        <template #body="{ data }">
          <Checkbox v-model="data.activeBridge" binary @change="onToggleBridge(data)" />
        </template>
      </Column>
      <Column header="IP assignment">
        <template #body="{ data }">
          <span class="zp-mono">{{ (data.ipAssignments || []).join(', ') || '—' }}</span>
        </template>
      </Column>
      <Column header="Status" style="width: 150px">
        <template #body="{ data }">
          <StatusPill :peer="data.peer" :is-controller="data.address === controllerAddress" />
        </template>
      </Column>
      <Column header="" style="width: 50px">
        <template #body="{ data }">
          <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Remove member" @click="confirmDelete(data)" />
        </template>
      </Column>
    </DataTable>
  </div>
</template>
