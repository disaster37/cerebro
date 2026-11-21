<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Nodes</h4>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="neutral" variant="ghost" :icon="chartsOpen ? 'i-lucide-chevron-up' : 'i-lucide-chart-bar'" @click="chartsOpen = !chartsOpen">
          Charts
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Refresh
        </UButton>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="nodes-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else>
      <UCard v-if="chartsOpen" class="mb-6">
        <template #header>
          <h5 class="text-sm font-semibold">Node comparison</h5>
        </template>
        <div class="h-72">
          <Bar :key="`nodes-${colorMode.value}`" :data="chartData" :options="nodeBarOptions" />
        </div>
      </UCard>

      <CerebroTable
        :data="filteredNodes"
        :columns="columns"
        :default-sort="[{ id: 'name', desc: false }]"
        filter-placeholder="Filter nodes..."
        empty-text="No nodes found."
      >
        <template #filters>
          <UCheckbox v-model="roleFilter.M" label="Master" />
          <UCheckbox v-model="roleFilter.D" label="Data" />
          <UCheckbox v-model="roleFilter.I" label="Ingest" />
          <UCheckbox v-model="roleFilter.C" label="Coordinating" />
        </template>
        <template #name-cell="{ row }">
          <div class="flex items-center gap-1">
            <UIcon
              name="i-lucide-star"
              :class="row.original.current_master ? 'text-amber-500' : 'text-neutral-400'"
              :style="row.original.current_master ? '' : 'opacity:.35'"
            />
            <button class="font-medium hover:underline" @click="inspectNode(row.original)">{{ row.original.name }}</button>
          </div>
          <div class="text-[11px] text-neutral-500">{{ row.original.host }} / {{ row.original.ip }}</div>
        </template>
        <template #role-cell="{ row }">
          <div class="flex gap-1">
            <UBadge v-if="row.original.master" color="primary" variant="soft">M</UBadge>
            <UBadge v-if="row.original.data" color="success" variant="soft">D</UBadge>
            <UBadge v-if="row.original.ingest" color="info" variant="soft">I</UBadge>
            <UBadge v-if="row.original.coordinating" color="neutral" variant="soft">C</UBadge>
          </div>
        </template>
        <template #load-cell="{ row }">
          <div>{{ loadLabel(row.original) }}</div>
          <div class="text-[11px] text-neutral-500">
            5m: {{ row.original.load_average_5m != null ? row.original.load_average_5m.toFixed(2) : 'N/A' }} /
            15m: {{ row.original.load_average_15m != null ? row.original.load_average_15m.toFixed(2) : 'N/A' }}
            (cores: {{ row.original.available_processors }})
          </div>
        </template>
        <template #cpu-cell="{ row }">
          <UTooltip :text="cpuTooltip(row.original)">
            <span :class="effectiveCpuOf(row.original) === null ? 'text-neutral-400' : ''">{{ formatPercent(effectiveCpuOf(row.original)) }}</span>
          </UTooltip>
          <div class="text-[11px] text-neutral-500">{{ cpuSubtext(row.original) }}</div>
          <UProgress
            :model-value="Math.min(effectiveCpuOf(row.original) ?? 0, 100)"
            size="xs"
            :color="effectiveCpuOf(row.original) !== null ? 'info' : 'neutral'"
            :class="{ 'opacity-35': effectiveCpuOf(row.original) === null }"
          />
        </template>
        <template #heap-cell="{ row }">
          <div>{{ row.original.heap?.used_percent ?? 0 }}%</div>
          <div class="text-[11px] text-neutral-500">{{ formatBytes(row.original.heap?.used) }} / {{ formatBytes(row.original.heap?.max) }}</div>
          <UProgress :model-value="row.original.heap?.used_percent ?? 0" size="xs" color="warning" />
        </template>
        <template #disk-cell="{ row }">
          <div>{{ row.original.disk?.used_percent ?? 0 }}%</div>
          <div class="text-[11px] text-neutral-500">free: {{ formatBytes(row.original.disk?.free) }} / {{ formatBytes(row.original.disk?.total) }}</div>
          <UProgress :model-value="row.original.disk?.used_percent ?? 0" size="xs" color="error" />
        </template>
        <template #uptime-cell="{ row }">
          {{ formatDuration(row.original.uptime_in_millis) }}
        </template>
        <template #es_version-cell="{ row }">
          <div>{{ row.original.es_version }}</div>
          <div class="text-[11px] text-neutral-500">JVM {{ row.original.jvm_version }}</div>
        </template>
      </CerebroTable>
    </div>

    <JsonInspectorModal v-model:open="inspectorOpen" :title="inspectorTitle" :data="inspectorData" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Bar } from 'vue-chartjs'
import CerebroTable from '../../components/CerebroTable.vue'
import JsonInspectorModal from '../../components/JsonInspectorModal.vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import { useChartTheme } from '../../composables/useChartTheme'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'
import type { NodeCpuSource, NodeEntry } from '../../types/cerebro'
import { formatBytes, formatDuration, formatPercent, normalizePercent } from '../../utils/format'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const { tokens, barOptions } = useChartTheme()
const colorMode = useColorMode()

const nodes = ref<NodeEntry[]>([])
const loading = ref(true)
const error = ref('')
const chartsOpen = ref(false)
const roleFilter = ref<Record<string, boolean>>({ M: true, D: true, I: true, C: true })
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)

const filteredNodes = computed(() => nodes.value.filter((n) => {
  const hasRole = n.master || n.data || n.ingest || n.coordinating
  if (!hasRole) {
    return roleFilter.value.M && roleFilter.value.D && roleFilter.value.I && roleFilter.value.C
  }
  return (n.master && roleFilter.value.M)
    || (n.data && roleFilter.value.D)
    || (n.ingest && roleFilter.value.I)
    || (n.coordinating && roleFilter.value.C)
}))

function effectiveCpuOf(node: NodeEntry): number | null {
  return normalizePercent(node.cpu_percent)
    ?? normalizePercent(node.resource_cpu_percent)
    ?? normalizePercent(node.os_cpu_percent)
    ?? normalizePercent(node.process_cpu_percent)
}

function cpuSourceOf(node: NodeEntry): NodeCpuSource | undefined {
  if (node.cpu_source) return node.cpu_source
  if (normalizePercent(node.resource_cpu_percent) !== null) return 'resource_tracker'
  if (normalizePercent(node.cpu_percent) !== null || normalizePercent(node.os_cpu_percent) !== null) return 'os'
  if (normalizePercent(node.process_cpu_percent) !== null) return 'process'
  return undefined
}

function cpuSubtext(node: NodeEntry): string {
  const source = cpuSourceOf(node)
  if (source === 'resource_tracker') return 'process CPU (resource tracker)'
  if (source === 'process') return 'process CPU (OS: N/A)'
  return `process: ${formatPercent(node.process_cpu_percent)}`
}

function cpuTooltip(node: NodeEntry): string {
  const source = cpuSourceOf(node)
  const label = source === 'resource_tracker'
    ? 'resource tracker — process CPU window avg'
    : source === 'os'
      ? 'OS CPU'
      : source === 'process'
        ? 'JVM process CPU'
        : 'unavailable'
  return `Effective: ${formatPercent(effectiveCpuOf(node))} (${label}) | OS CPU: ${formatPercent(node.os_cpu_percent)} | JVM process CPU: ${formatPercent(node.process_cpu_percent)}`
}

const chartData = computed(() => {
  const cpuSeries = filteredNodes.value.map((n) => effectiveCpuOf(n))
  const procSeries = filteredNodes.value.map((n) => normalizePercent(n.process_cpu_percent))
  const datasets: any[] = []
  if (cpuSeries.some((v) => v !== null)) {
    datasets.push({ label: 'CPU %', data: cpuSeries, backgroundColor: tokens.value.series.replica })
  }
  if (procSeries.some((v) => v !== null)) {
    datasets.push({ label: 'JVM Process CPU %', data: procSeries, backgroundColor: tokens.value.series.relocating })
  }
  datasets.push(
    { label: 'Heap %', data: filteredNodes.value.map((n) => n.heap?.used_percent ?? 0), backgroundColor: tokens.value.series.warning },
    { label: 'Disk %', data: filteredNodes.value.map((n) => n.disk?.used_percent ?? 0), backgroundColor: tokens.value.series.danger },
  )
  return { labels: filteredNodes.value.map((n) => n.name), datasets }
})

const nodeBarOptions = computed(() => ({
  ...barOptions.value,
  plugins: {
    ...barOptions.value.plugins,
    legend: {
      display: true,
      position: 'bottom' as const,
      labels: { color: tokens.value.textColor, boxWidth: 10, boxHeight: 10 },
    },
  },
}))

const columns: CerebroColumnDef<NodeEntry>[] = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'role', header: 'Role', accessorFn: (n) => `${n.master ? 'M' : ''}${n.data ? 'D' : ''}${n.ingest ? 'I' : ''}${n.coordinating ? 'C' : ''}`, class: 'whitespace-nowrap' },
  { id: 'load', header: 'Load', accessorFn: (n) => (n.load_average !== null && n.load_average !== undefined && n.load_average >= 0 ? n.load_average : undefined), sortUndefined: 'last' },
  { id: 'cpu', header: 'CPU', accessorFn: (n) => effectiveCpuOf(n) ?? undefined, sortUndefined: 'last', class: 'min-w-28' },
  { id: 'heap', header: 'Heap', accessorFn: (n) => n.heap?.used_percent ?? 0, class: 'min-w-28' },
  { id: 'disk', header: 'Disk', accessorFn: (n) => n.disk?.used_percent ?? 0, class: 'min-w-28' },
  { id: 'uptime', header: 'Uptime', accessorFn: (n) => n.uptime_in_millis ?? 0, class: 'whitespace-nowrap' },
  { id: 'es_version', header: 'Version', accessorKey: 'es_version', class: 'whitespace-nowrap' },
]

function loadLabel(node: NodeEntry) {
  if (node.load_average === null || node.load_average === undefined) return 'N/A'
  return node.load_average.toFixed(2)
}

async function inspectNode(node: NodeEntry) {
  const resp = await apiCall('POST', '/commons/get_node_stats', { host: clusterHost.value, node: node.id })
  inspectorTitle.value = `Node stats: ${node.name}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/nodes', { host: clusterHost.value }, { suppressErrorAlert: true })
  if (resp?.status === 200 && Array.isArray(resp.body)) {
    nodes.value = resp.body
    error.value = ''
  } else if (resp?.status === 401 || resp?.status === 403) {
    error.value = `Unauthorized (${resp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load cluster nodes (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load cluster nodes (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

onMounted(load)
</script>
