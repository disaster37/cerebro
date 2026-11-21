<script setup lang="ts">
import { computed } from 'vue'
import type { OverviewNode } from '../../types/cerebro'
import { formatBytes, formatPercent, normalizePercent } from '../../utils/format'

const props = defineProps<{ node: OverviewNode }>()

const heapPercent = computed(() => Math.round(props.node.heap?.used_percent || 0))
const diskPercent = computed(() => Math.round(props.node.disk?.used_percent || 0))
const cpuValue = computed(() => normalizePercent(props.node.cpu_percent)
  ?? normalizePercent(props.node.resource_cpu_percent)
  ?? normalizePercent(props.node.process_cpu_percent))
const cpuTooltip = computed(() => {
  if (props.node.cpu_source === 'resource_tracker') {
    return `CPU ${formatPercent(cpuValue.value)} (process CPU — resource tracker)`
  }
  if (props.node.cpu_source === 'process') {
    return `CPU ${formatPercent(cpuValue.value)} (process CPU)`
  }
  if (cpuValue.value !== null) {
    return `CPU ${formatPercent(cpuValue.value)}`
  }
  return 'CPU N/A'
})
const loadLabel = computed(() => {
  if (props.node.load_average === null || props.node.load_average === undefined) return 'N/A'
  return props.node.load_average.toFixed(2)
})
const loadPercent = computed(() => {
  const load = props.node.load_average
  const cores = props.node.available_processors || 1
  if (load === null || load === undefined) return 0
  return Math.min(Math.round((load / cores) * 100), 100)
})
</script>

<template>
  <div class="grid w-full grid-cols-4 gap-1.5">
    <UTooltip :text="`Heap ${heapPercent}% (${formatBytes(node.heap?.used)} / ${formatBytes(node.heap?.max)})`">
      <div class="min-w-0">
        <UProgress :model-value="heapPercent" size="xs" color="warning" />
      </div>
    </UTooltip>
    <UTooltip :text="`Disk ${diskPercent}% (free ${formatBytes(node.disk?.free)} / ${formatBytes(node.disk?.total)})`">
      <div class="min-w-0">
        <UProgress :model-value="diskPercent" size="xs" color="error" />
      </div>
    </UTooltip>
    <UTooltip :text="cpuTooltip">
      <div class="min-w-0">
        <UProgress
          :model-value="cpuValue !== null ? Math.min(Math.round(cpuValue), 100) : 0"
          size="xs"
          :color="cpuValue !== null ? 'info' : 'neutral'"
          :class="{ 'opacity-35': cpuValue === null }"
        />
      </div>
    </UTooltip>
    <UTooltip :text="`Load ${loadLabel}`">
      <div class="min-w-0">
        <UProgress :model-value="loadPercent" size="xs" color="success" />
      </div>
    </UTooltip>
  </div>
</template>
