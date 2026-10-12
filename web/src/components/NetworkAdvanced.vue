<script setup lang="ts">
import Panel from 'primevue/panel'

defineProps<{ network: Record<string, any> }>()
</script>

<template>
  <Panel header="Advanced: raw controller data" toggleable collapsed>
    <p class="zp-page-subtitle" style="margin-top: 0">
      Everything the ZeroTier controller knows about this network, unfiltered. You shouldn't need this
      for day-to-day use -- it's here for debugging.
    </p>
    <pre class="raw-json">{{ JSON.stringify(network, null, 2) }}</pre>
  </Panel>
</template>

<style scoped>
.raw-json {
  background: #0f172a;
  color: #e2e8f0;
  padding: 16px;
  border-radius: 8px;
  font-family: var(--zp-font-mono);
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  max-height: 420px;
}
</style>
