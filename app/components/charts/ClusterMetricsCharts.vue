<template>
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
    <UCard :class="tokens.cardClass">
      <template #header>
        <h4 class="text-sm font-semibold">Shard Health</h4>
      </template>
      <div class="h-48">
        <Doughnut :key="`shards-${colorMode.value}`" :data="shardData" :options="doughnutOptions" />
      </div>
    </UCard>

    <UCard :class="tokens.cardClass">
      <template #header>
        <h4 class="text-sm font-semibold">Cluster JVM Heap</h4>
      </template>
      <div class="h-48">
        <Doughnut :key="`heap-${colorMode.value}`" :data="heapData" :options="doughnutOptions" />
      </div>
    </UCard>

    <UCard :class="tokens.cardClass">
      <template #header>
        <h4 class="text-sm font-semibold">Cluster Disk</h4>
      </template>
      <div class="h-48">
        <Doughnut :key="`disk-${colorMode.value}`" :data="diskData" :options="doughnutOptions" />
      </div>
    </UCard>

    <UCard :class="[tokens.cardClass, 'lg:col-span-3']">
      <template #header>
        <h4 class="text-sm font-semibold">Top Indices by Size</h4>
      </template>
      <div class="h-64">
        <Bar :key="`indices-${colorMode.value}`" :data="topIndicesData" :options="barOptions" />
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Bar, Doughnut } from 'vue-chartjs'
import type { OverviewData } from '../../types/cerebro'
import { useChartTheme } from '../../composables/useChartTheme'

const props = defineProps<{ data: OverviewData }>()

const colorMode = useColorMode()
const { tokens, doughnutOptions, barOptions } = useChartTheme()

const shardData = computed(() => ({
  labels: ['Started', 'Initializing', 'Relocating', 'Unassigned'],
  datasets: [{
    data: [
      Math.max((props.data.active_shards || 0) - (props.data.initializing_shards || 0) - (props.data.relocating_shards || 0), 0),
      props.data.initializing_shards || 0,
      props.data.relocating_shards || 0,
      props.data.unassigned_shards || 0,
    ],
    backgroundColor: [
      tokens.value.series.healthy,
      tokens.value.series.replica,
      tokens.value.series.relocating,
      tokens.value.series.warning,
    ],
    borderColor: tokens.value.borderColor,
    borderWidth: 2,
  }],
}))

const heapData = computed(() => {
  const used = (props.data.nodes || []).reduce((sum, n) => sum + (n.heap?.used || 0), 0)
  const max = (props.data.nodes || []).reduce((sum, n) => sum + (n.heap?.max || 0), 0)
  return {
    labels: ['Used', 'Free'],
    datasets: [{
      data: [used, Math.max(max - used, 0)],
      backgroundColor: [tokens.value.series.warning, tokens.value.series.track],
      borderColor: tokens.value.borderColor,
      borderWidth: 2,
    }],
  }
})

const diskData = computed(() => {
  const total = (props.data.nodes || []).reduce((sum, n) => sum + (n.disk?.total || 0), 0)
  const free = (props.data.nodes || []).reduce((sum, n) => sum + (n.disk?.free || 0), 0)
  return {
    labels: ['Used', 'Free'],
    datasets: [{
      data: [Math.max(total - free, 0), free],
      backgroundColor: [tokens.value.series.danger, tokens.value.series.track],
      borderColor: tokens.value.borderColor,
      borderWidth: 2,
    }],
  }
})

const topIndicesData = computed(() => {
  const top = [...(props.data.indices || [])]
    .sort((a, b) => (b.total_size_in_bytes || 0) - (a.total_size_in_bytes || 0))
    .slice(0, 10)
  return {
    labels: top.map((i) => i.name),
    datasets: [{
      label: 'Total size',
      data: top.map((i) => i.total_size_in_bytes || 0),
      backgroundColor: tokens.value.series.replica,
      borderRadius: 4,
    }],
  }
})
</script>
