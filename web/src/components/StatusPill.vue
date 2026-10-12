<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  peer?: { latency?: number; versionMajor?: number; version?: string } | null
  isController?: boolean
}>()

const status = computed(() => {
  if (props.isController) return { label: 'Controller', cls: 'controller' }
  if (!props.peer || props.peer.versionMajor === -1) return { label: 'Offline', cls: 'offline' }
  if (props.peer.latency === -1) return { label: 'Relay', cls: 'relay' }
  return { label: `Online · ${props.peer.latency}ms`, cls: 'online' }
})
</script>

<template>
  <span class="pill" :class="status.cls">{{ status.label }}</span>
</template>

<style scoped>
.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}
.pill::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.online { background: #eafbf0; color: #16a34a; }
.relay { background: #fff7e6; color: #d97706; }
.offline { background: #fdecec; color: #dc2626; }
.controller { background: #eaf1ff; color: #2563eb; }
</style>
