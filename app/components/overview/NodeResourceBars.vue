<template>
  <div class="flex items-center gap-2">
    <UTooltip :text="`Heap ${heapPercent}% (${formatBytes(node.heap?.used)} / ${formatBytes(node.heap?.max)})`">
      <div class="w-16">
        <UProgress :model-value="heapPercent" size="xs" color="warning" />
      </div>
    </UTooltip>
    <UTooltip :text="`Disk ${diskPercent}% (free ${formatBytes(node.disk?.free)} / ${formatBytes(node.disk?.total)})`">
      <div class="w-16">
        <UProgress :model-value="diskPercent" size="xs" color="error" />
      </div>
    </UTooltip>
    <UTooltip :text="`CPU ${cpuPercent}%`">
      <div class="w-16">
        <UProgress :model-value="cpuPercent" size="xs" color="info" />
      </div>
    </UTooltip>
    <UTooltip :text="`Load ${loadLabel}`">
      <div class="w-16">
        <UProgress :model-value="loadPercent" size="xs" color="success" />
      </div>
    </UTooltip>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { OverviewNode } from '../../types/cerebro'
import { formatBytes } from '../../utils/format'

const props = defineProps<{ node: OverviewNode }>()

const heapPercent = computed(() => Math.round(props.node.heap?.used_percent || 0))
const diskPercent = computed(() => Math.round(props.node.disk?.used_percent || 0))
const cpuPercent = computed(() => Math.round(props.node.cpu_percent || 0))
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
