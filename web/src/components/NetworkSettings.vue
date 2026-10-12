<script setup lang="ts">
import { computed, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { api, errorMessage } from '../api/client'
import Accordion from 'primevue/accordion'
import AccordionPanel from 'primevue/accordionpanel'
import AccordionHeader from 'primevue/accordionheader'
import AccordionContent from 'primevue/accordioncontent'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Textarea from 'primevue/textarea'

interface Route { target: string; via: string | null }
interface IpPool { ipRangeStart: string; ipRangeEnd: string }

const props = defineProps<{ networkId: string; network: Record<string, any> }>()
const emit = defineEmits<{ (e: 'updated'): void }>()

const toast = useToast()
const saving = ref(false)

async function patchNetwork(fields: Record<string, unknown>, successMessage?: string) {
  saving.value = true
  try {
    await api.patch(`/networks/${props.networkId}`, fields)
    emit('updated')
    if (successMessage) toast.add({ severity: 'success', summary: successMessage, life: 2500 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Could not save change', detail: errorMessage(err), life: 5000 })
  } finally {
    saving.value = false
  }
}

// ---- Routes ----
const newRouteTarget = ref('')
const newRouteVia = ref('')

function addRoute() {
  if (!newRouteTarget.value) return
  const routes: Route[] = [...(props.network.routes || []), { target: newRouteTarget.value, via: newRouteVia.value || null }]
  patchNetwork({ routes }, 'Route added')
  newRouteTarget.value = ''
  newRouteVia.value = ''
}

function removeRoute(route: Route) {
  const routes = (props.network.routes || []).filter((r: Route) => r.target !== route.target)
  patchNetwork({ routes }, 'Route removed')
}

// ---- IP assignment pools ----
const newPoolStart = ref('')
const newPoolEnd = ref('')

function addPool() {
  if (!newPoolStart.value || !newPoolEnd.value) return
  const ipAssignmentPools: IpPool[] = [
    ...(props.network.ipAssignmentPools || []),
    { ipRangeStart: newPoolStart.value, ipRangeEnd: newPoolEnd.value },
  ]
  patchNetwork({ ipAssignmentPools }, 'IP pool added')
  newPoolStart.value = ''
  newPoolEnd.value = ''
}

function removePool(pool: IpPool) {
  const ipAssignmentPools = (props.network.ipAssignmentPools || []).filter(
    (p: IpPool) => !(p.ipRangeStart === pool.ipRangeStart && p.ipRangeEnd === pool.ipRangeEnd),
  )
  patchNetwork({ ipAssignmentPools }, 'IP pool removed')
}

// ---- Assign modes ----
const v4Auto = computed({
  get: () => !!props.network.v4AssignMode?.zt,
  set: (val) => patchNetwork({ v4AssignMode: { zt: val } }),
})

const v6Modes = computed(() => props.network.v6AssignMode || {})
function setV6Mode(key: string, value: boolean) {
  patchNetwork({ v6AssignMode: { ...v6Modes.value, [key]: value } })
}

// ---- DNS ----
const dnsDomain = ref(props.network.dns?.domain || '')
const dnsServers = ref((props.network.dns?.servers || []).join('\n'))

function saveDns() {
  patchNetwork(
    { dns: { domain: dnsDomain.value, servers: dnsServers.value.split('\n').map((s: string) => s.trim()).filter(Boolean) } },
    'DNS settings saved',
  )
}

// ---- Auth tokens ----
// The original ztncui never implemented editing this field (it's a
// real part of the controller API: tokens that let a device join a
// private network without manual authorization). ztpanel treats it
// as an ordinary editable list, like routes or IP pools.
const newAuthToken = ref('')

// The controller returns `[null]` rather than `[]` when no tokens are
// set -- filter that out so it doesn't render as a blank row.
const validAuthTokens = computed<string[]>(() => (props.network.authTokens || []).filter((t: unknown) => !!t))

function addAuthToken() {
  if (!newAuthToken.value) return
  const authTokens: string[] = [...validAuthTokens.value, newAuthToken.value]
  patchNetwork({ authTokens }, 'Auth token added')
  newAuthToken.value = ''
}

function removeAuthToken(token: string) {
  const authTokens = validAuthTokens.value.filter((t: string) => t !== token)
  patchNetwork({ authTokens }, 'Auth token removed')
}
</script>

<template>
  <Accordion :value="['routes']" multiple>
    <AccordionPanel value="routes">
      <AccordionHeader>Routes</AccordionHeader>
      <AccordionContent>
        <DataTable :value="network.routes || []" size="small">
          <Column field="target" header="Target" />
          <Column field="via" header="Gateway">
            <template #body="{ data }">{{ data.via || '(this network)' }}</template>
          </Column>
          <Column header="" style="width: 44px">
            <template #body="{ data }">
              <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Remove route" @click="removeRoute(data)" />
            </template>
          </Column>
          <template #empty>
            <span class="zp-page-subtitle">No routes configured.</span>
          </template>
        </DataTable>
        <div class="zp-row" style="margin-top: 14px; align-items: flex-end">
          <div class="zp-field" style="flex: 1; margin-bottom: 0">
            <label>Target (CIDR)</label>
            <InputText v-model="newRouteTarget" placeholder="10.11.12.0/24" fluid />
          </div>
          <div class="zp-field" style="flex: 1; margin-bottom: 0">
            <label>Gateway (optional)</label>
            <InputText v-model="newRouteVia" placeholder="172.16.2.1" fluid />
          </div>
          <Button label="Add" icon="pi pi-plus" @click="addRoute" />
        </div>
      </AccordionContent>
    </AccordionPanel>

    <AccordionPanel value="pools">
      <AccordionHeader>IP assignment pools</AccordionHeader>
      <AccordionContent>
        <DataTable :value="network.ipAssignmentPools || []" size="small">
          <Column field="ipRangeStart" header="Range start" />
          <Column field="ipRangeEnd" header="Range end" />
          <Column header="" style="width: 44px">
            <template #body="{ data }">
              <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Remove IP pool" @click="removePool(data)" />
            </template>
          </Column>
          <template #empty>
            <span class="zp-page-subtitle">No IP assignment pools configured.</span>
          </template>
        </DataTable>
        <div class="zp-row" style="margin-top: 14px; align-items: flex-end">
          <div class="zp-field" style="flex: 1; margin-bottom: 0">
            <label>Range start</label>
            <InputText v-model="newPoolStart" placeholder="10.11.12.1" fluid />
          </div>
          <div class="zp-field" style="flex: 1; margin-bottom: 0">
            <label>Range end</label>
            <InputText v-model="newPoolEnd" placeholder="10.11.12.254" fluid />
          </div>
          <Button label="Add" icon="pi pi-plus" @click="addPool" />
        </div>
      </AccordionContent>
    </AccordionPanel>

    <AccordionPanel value="assign">
      <AccordionHeader>Address assignment</AccordionHeader>
      <AccordionContent>
        <div class="zp-row" style="margin-bottom: 10px">
          <Checkbox v-model="v4Auto" binary input-id="v4auto" />
          <label for="v4auto">IPv4: auto-assign from the pool above</label>
        </div>
        <div class="zp-row" style="margin-bottom: 10px">
          <Checkbox :model-value="!!v6Modes['6plane']" binary @update:model-value="(v) => setV6Mode('6plane', v)" input-id="v6plane" />
          <label for="v6plane">IPv6: ZT 6PLANE (routable /80 per device)</label>
        </div>
        <div class="zp-row" style="margin-bottom: 10px">
          <Checkbox :model-value="!!v6Modes['rfc4193']" binary @update:model-value="(v) => setV6Mode('rfc4193', v)" input-id="v6rfc" />
          <label for="v6rfc">IPv6: RFC4193 (/128 per device)</label>
        </div>
        <div class="zp-row">
          <Checkbox :model-value="!!v6Modes['zt']" binary @update:model-value="(v) => setV6Mode('zt', v)" input-id="v6zt" />
          <label for="v6zt">IPv6: auto-assign from the pool above</label>
        </div>
      </AccordionContent>
    </AccordionPanel>

    <AccordionPanel value="dns">
      <AccordionHeader>DNS</AccordionHeader>
      <AccordionContent>
        <div class="zp-field">
          <label>Search domain</label>
          <InputText v-model="dnsDomain" fluid placeholder="cartorios.internal" />
        </div>
        <div class="zp-field">
          <label>Servers (one per line)</label>
          <Textarea v-model="dnsServers" fluid rows="3" placeholder="10.11.12.1" />
        </div>
        <Button label="Save DNS settings" :loading="saving" @click="saveDns" />
      </AccordionContent>
    </AccordionPanel>

    <AccordionPanel value="authtokens">
      <AccordionHeader>Auth tokens</AccordionHeader>
      <AccordionContent>
        <p class="zp-page-subtitle" style="margin-top: 0">
          Tokens that let a device join this network without manual authorization.
        </p>
        <DataTable :value="validAuthTokens.map((t: string) => ({ token: t }))" size="small">
          <Column field="token" header="Token">
            <template #body="{ data }"><span class="zp-mono">{{ data.token }}</span></template>
          </Column>
          <Column header="" style="width: 44px">
            <template #body="{ data }">
              <Button icon="pi pi-trash" severity="danger" text rounded aria-label="Remove auth token" @click="removeAuthToken(data.token)" />
            </template>
          </Column>
          <template #empty>
            <span class="zp-page-subtitle">No auth tokens configured.</span>
          </template>
        </DataTable>
        <div class="zp-row" style="margin-top: 14px; align-items: flex-end">
          <div class="zp-field" style="flex: 1; margin-bottom: 0">
            <label>New token</label>
            <InputText v-model="newAuthToken" fluid placeholder="a shared secret" />
          </div>
          <Button label="Add" icon="pi pi-plus" @click="addAuthToken" />
        </div>
      </AccordionContent>
    </AccordionPanel>
  </Accordion>
</template>
