<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { api, errorMessage } from '../api/client'
import { useAuthStore } from '../stores/auth'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import Skeleton from 'primevue/skeleton'

interface OrgMember {
  username: string
  role: 'ADMIN' | 'USER'
}

interface Organization {
  id: string
  name: string
  members: OrgMember[]
  networkIds: string[]
}

interface NetworkSummary {
  nwid: string
  name: string
  private: boolean
  memberCount: number
}

const props = defineProps<{ id: string }>()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()
const auth = useAuthStore()

const org = ref<Organization | null>(null)
const networks = ref<NetworkSummary[]>([])
const loading = ref(true)

const myRole = computed(() => org.value?.members.find((m) => m.username === auth.username)?.role)
const isAdmin = computed(() => myRole.value === 'ADMIN')

async function load() {
  loading.value = true
  try {
    const [{ data: orgData }, { data: netData }] = await Promise.all([
      api.get<Organization>(`/organizations/${props.id}`),
      api.get<NetworkSummary[]>(`/organizations/${props.id}/networks`),
    ])
    org.value = orgData
    networks.value = netData
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not load organization', detail: errorMessage(err), life: 5000 })
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- Create + assign network ----
const createNetOpen = ref(false)
const newNetName = ref('')
const creatingNet = ref(false)

async function createNetwork() {
  creatingNet.value = true
  try {
    const { data: net } = await api.post('/networks', { name: newNetName.value || 'New network' })
    await api.post(`/organizations/${props.id}/networks/${net.nwid}`)
    createNetOpen.value = false
    newNetName.value = ''
    await load()
    toast.add({ severity: 'success', summary: 'Network created', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not create network', detail: errorMessage(err), life: 5000 })
  } finally {
    creatingNet.value = false
  }
}

function openNetwork(network: NetworkSummary) {
  router.push({ name: 'network-detail', params: { id: network.nwid } })
}

function removeFromOrg(network: NetworkSummary) {
  confirm.require({
    message: `Remove "${network.name}" from this organization? The network itself is not deleted.`,
    header: 'Remove network',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Remove' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      await api.delete(`/organizations/${props.id}/networks/${network.nwid}`)
      await load()
    },
  })
}

// ---- Members ----
const addMemberOpen = ref(false)
const newMemberName = ref('')
const newMemberRole = ref<'ADMIN' | 'USER'>('USER')
const addingMember = ref(false)
const roleOptions = [
  { label: 'Admin', value: 'ADMIN' },
  { label: 'User', value: 'USER' },
]

async function addMember() {
  if (!newMemberName.value) return
  addingMember.value = true
  try {
    await api.post(`/organizations/${props.id}/members`, { username: newMemberName.value, role: newMemberRole.value })
    addMemberOpen.value = false
    newMemberName.value = ''
    await load()
    toast.add({ severity: 'success', summary: 'Member added', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not add member', detail: errorMessage(err), life: 5000 })
  } finally {
    addingMember.value = false
  }
}

function removeMember(member: OrgMember) {
  confirm.require({
    message: `Remove ${member.username} from this organization?`,
    header: 'Remove member',
    icon: 'pi pi-exclamation-triangle',
    acceptProps: { severity: 'danger', label: 'Remove' },
    rejectProps: { severity: 'secondary', outlined: true, label: 'Cancel' },
    accept: async () => {
      await api.delete(`/organizations/${props.id}/members/${member.username}`)
      await load()
    },
  })
}
</script>

<template>
  <div class="zp-page">
    <div v-if="loading">
      <Skeleton height="2.4rem" style="margin-bottom: 14px" />
      <Skeleton height="10rem" />
    </div>

    <template v-else-if="org">
      <div class="zp-page-header">
        <div>
          <h1>{{ org.name }}</h1>
          <p class="zp-page-subtitle">{{ org.members.length }} members · {{ networks.length }} networks</p>
        </div>
      </div>

      <Tabs value="networks">
        <TabList>
          <Tab value="networks">Networks</Tab>
          <Tab value="members">Members</Tab>
        </TabList>
        <TabPanels>
          <TabPanel value="networks">
            <div class="zp-row-between" style="margin-bottom: 12px">
              <span />
              <Button v-if="isAdmin" label="Add network" icon="pi pi-plus" size="small" @click="createNetOpen = true" />
            </div>
            <div class="zp-card" style="padding: 0">
              <div v-if="networks.length === 0" class="zp-empty">
                <i class="pi pi-sitemap" style="font-size: 26px; opacity: 0.4" />
                <p>No networks in this organization yet.</p>
              </div>
              <DataTable v-else :value="networks" @row-click="(e) => openNetwork(e.data)" row-hover style="cursor: pointer">
                <Column field="name" header="Name" />
                <Column field="nwid" header="Network ID">
                  <template #body="{ data }"><span class="zp-mono">{{ data.nwid }}</span></template>
                </Column>
                <Column field="private" header="Access">
                  <template #body="{ data }">
                    <Tag :severity="data.private ? 'info' : 'warn'" :value="data.private ? 'Private' : 'Public'" />
                  </template>
                </Column>
                <Column field="memberCount" header="Members" />
                <Column v-if="isAdmin" header="" style="width: 50px">
                  <template #body="{ data }">
                    <Button icon="pi pi-times" text rounded aria-label="Remove from organization" @click.stop="removeFromOrg(data)" />
                  </template>
                </Column>
              </DataTable>
            </div>
          </TabPanel>

          <TabPanel value="members">
            <div class="zp-row-between" style="margin-bottom: 12px">
              <span />
              <Button v-if="isAdmin" label="Add member" icon="pi pi-plus" size="small" @click="addMemberOpen = true" />
            </div>
            <div class="zp-card" style="padding: 0">
              <DataTable :value="org.members">
                <Column field="username" header="Username" />
                <Column field="role" header="Role">
                  <template #body="{ data }">
                    <Tag :severity="data.role === 'ADMIN' ? 'info' : 'secondary'" :value="data.role" />
                  </template>
                </Column>
                <Column v-if="isAdmin" header="" style="width: 50px">
                  <template #body="{ data }">
                    <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Remove member" @click="removeMember(data)" />
                  </template>
                </Column>
              </DataTable>
            </div>
          </TabPanel>
        </TabPanels>
      </Tabs>
    </template>

    <Dialog v-model:visible="createNetOpen" header="Add network" modal style="width: 420px">
      <div class="zp-field">
        <label for="newNetName">Network name</label>
        <InputText id="newNetName" v-model="newNetName" fluid autofocus />
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Cancel" severity="secondary" outlined @click="createNetOpen = false" />
        <Button label="Create" :loading="creatingNet" @click="createNetwork" />
      </div>
    </Dialog>

    <Dialog v-model:visible="addMemberOpen" header="Add member" modal style="width: 420px">
      <div class="zp-field">
        <label for="newMemberName">Username</label>
        <InputText id="newMemberName" v-model="newMemberName" fluid autofocus placeholder="must already have a ztpanel account" />
      </div>
      <div class="zp-field">
        <label for="newMemberRole">Role</label>
        <Select id="newMemberRole" v-model="newMemberRole" :options="roleOptions" option-label="label" option-value="value" fluid />
      </div>
      <div class="zp-row" style="justify-content: flex-end; margin-top: 10px">
        <Button label="Cancel" severity="secondary" outlined @click="addMemberOpen = false" />
        <Button label="Add" :loading="addingMember" @click="addMember" />
      </div>
    </Dialog>
  </div>
</template>
