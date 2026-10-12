<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Checkbox from 'primevue/checkbox'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'
import NetworkMembers from '../components/NetworkMembers.vue'
import NetworkSettings from '../components/NetworkSettings.vue'
import NetworkAdvanced from '../components/NetworkAdvanced.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()

const network = ref<Record<string, any> | null>(null)
const controllerAddress = ref<string>()
const loading = ref(true)
const savingName = ref(false)
const editingName = ref(false)
const nameDraft = ref('')

async function load() {
  loading.value = true
  try {
    const [{ data: net }, { data: status }] = await Promise.all([
      api.get(`/networks/${props.id}`),
      api.get('/status'),
    ])
    network.value = net
    controllerAddress.value = status.address
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load network', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function patch(fields: Record<string, unknown>, successMessage?: string) {
  try {
    const { data } = await api.patch(`/networks/${props.id}`, fields)
    network.value = data
    if (successMessage) toast.add({ severity: 'success', summary: successMessage, life: 2500 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not save change', detail: errorMessage(err), life: 5000 })
  }
}

function startEditName() {
  nameDraft.value = network.value?.name || ''
  editingName.value = true
}

async function saveName() {
  savingName.value = true
  await patch({ name: nameDraft.value })
  savingName.value = false
  editingName.value = false
}

function togglePrivate(value: boolean) {
  patch({ private: value }, value ? 'Network set to private' : 'Network set to public')
}

function confirmDeleteNetwork() {
  if (!network.value) return
  confirm.require({
    message: `Delete "${network.value.name}" (${network.value.nwid})? This cannot be undone.`,
    header: 'Delete network',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Delete' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      try {
        await api.delete(`/networks/${props.id}`)
        toast.add({ severity: 'success', summary: 'Network deleted', life: 3000 })
        router.push({ name: 'networks' })
      } catch (err) {
        toast.add({ severity: 'error', summary: 'Could not delete network', detail: errorMessage(err), life: 5000 })
      }
    },
  })
}
</script>

<template>
  <div class="zp-page">
    <Button label="Back to networks" icon="pi pi-arrow-left" text @click="router.push({ name: 'networks' })" style="margin-bottom: 10px; padding-left: 0" />

    <div v-if="loading">
      <Skeleton height="2.4rem" style="margin-bottom: 14px" />
      <Skeleton height="10rem" />
    </div>

    <template v-else-if="network">
      <div class="zp-page-header">
        <div>
          <div v-if="!editingName" class="zp-row" style="gap: 10px">
            <h1>{{ network.name || '(unnamed)' }}</h1>
            <Button icon="pi pi-pencil" text rounded size="small" aria-label="Rename network" @click="startEditName" />
          </div>
          <div v-else class="zp-row">
            <InputText v-model="nameDraft" size="small" @keyup.enter="saveName" />
            <Button icon="pi pi-check" :loading="savingName" rounded size="small" @click="saveName" />
            <Button icon="pi pi-times" text rounded size="small" @click="editingName = false" />
          </div>
          <p class="zp-page-subtitle zp-mono">{{ network.nwid }}</p>
        </div>
        <div class="zp-actions">
          <Tag :severity="network.private ? 'info' : 'warn'" :value="network.private ? 'Private' : 'Public'" />
          <Button label="Delete network" icon="pi pi-trash" severity="danger" outlined @click="confirmDeleteNetwork" />
        </div>
      </div>

      <Tabs value="overview">
        <TabList>
          <Tab value="overview">Overview</Tab>
          <Tab value="members">Members</Tab>
          <Tab value="settings">Settings</Tab>
          <Tab value="advanced">Advanced</Tab>
        </TabList>
        <TabPanels>
          <TabPanel value="overview">
            <div class="zp-card">
              <p class="zp-card-title">Access</p>
              <div class="zp-row">
                <Checkbox :model-value="!!network.private" binary input-id="privateToggle" @update:model-value="togglePrivate" />
                <label for="privateToggle">
                  Enable access control (private). Devices must be authorized to join.
                </label>
              </div>
            </div>

            <div class="zp-card">
              <p class="zp-card-title">Network parameters</p>
              <div class="zp-field">
                <label>MTU</label>
                <InputNumber :model-value="network.mtu" :use-grouping="false" fluid @update:model-value="(v) => patch({ mtu: v })" />
              </div>
              <div class="zp-field" style="margin-bottom: 0">
                <label>Multicast recipient limit</label>
                <InputNumber :model-value="network.multicastLimit" :use-grouping="false" fluid @update:model-value="(v) => patch({ multicastLimit: v })" />
              </div>
            </div>
          </TabPanel>

          <TabPanel value="members">
            <NetworkMembers :network-id="id" :controller-address="controllerAddress" />
          </TabPanel>

          <TabPanel value="settings">
            <NetworkSettings :network-id="id" :network="network" @updated="load" />
          </TabPanel>

          <TabPanel value="advanced">
            <NetworkAdvanced :network="network" />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </template>
  </div>
</template>
