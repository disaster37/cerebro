<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Indexes <small class="text-neutral-500">{{ clusterName }}</small></h4>
      <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
        Refresh
      </UButton>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="indices-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else-if="data">
      <div class="mb-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <UCard v-for="stat in stats" :key="stat.label" class="text-center">
          <p class="text-2xl font-semibold">{{ stat.value }}</p>
          <p class="text-xs text-neutral-500">{{ stat.label }}</p>
        </UCard>
      </div>

      <UCard class="mb-6">
        <template #header>
          <div class="flex items-center justify-between">
            <h5 class="text-sm font-semibold">Top 10 Indices</h5>
            <USelect v-model="chartMetric" :items="chartMetrics" size="xs" class="w-32" />
          </div>
        </template>
        <div class="h-64">
          <Bar :key="`indices-${colorMode.value}-${chartMetric}`" :data="chartData" :options="barOptions" />
        </div>
      </UCard>

      <CerebroTable
        :data="filteredIndices"
        :columns="columns"
        :default-sort="[{ id: 'name', desc: false }]"
        filter-placeholder="Filter indices..."
        empty-text="No indices found."
      >
        <template #filters>
          <UCheckbox v-model="showClosed" label="Show closed" />
          <UCheckbox v-model="showSpecial" label="Show .special" />
          <USelect v-model="healthFilter" :items="healthOptions" size="sm" class="w-32" />
        </template>
        <template #status-cell="{ row }">
          <UBadge v-if="row.original.closed" color="neutral" variant="soft">closed</UBadge>
          <UBadge v-else-if="row.original.unhealthy" color="error" variant="soft">unhealthy</UBadge>
          <UBadge v-else color="success" variant="soft">open</UBadge>
        </template>
        <template #shards-cell="{ row }">
          {{ row.original.num_shards }}P / {{ row.original.num_replicas }}R
        </template>
        <template #doc_count-cell="{ row }">
          {{ formatNumber(row.original.doc_count) }}
        </template>
        <template #deleted_docs-cell="{ row }">
          {{ formatNumber(row.original.deleted_docs) }}
        </template>
        <template #size_in_bytes-cell="{ row }">
          {{ formatBytes(row.original.size_in_bytes) }}
        </template>
        <template #total_size_in_bytes-cell="{ row }">
          {{ formatBytes(row.original.total_size_in_bytes) }}
        </template>
        <template #aliases-cell="{ row }">
          <span class="text-xs">{{ (row.original.aliases || []).join(', ') || '-' }}</span>
        </template>
        <template #actions-cell="{ row }">
          <UDropdownMenu :items="indexActions(row.original)">
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-ellipsis-vertical" aria-label="Index actions" />
          </UDropdownMenu>
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
import type { OverviewData, OverviewIndex } from '../../types/cerebro'
import { formatBytes, formatNumber } from '../../utils/format'

const { clusterHost, clusterName } = useCluster()
const { apiCall } = useAPI()
const { tokens, barOptions } = useChartTheme()
const colorMode = useColorMode()

const data = ref<OverviewData | null>(null)
const loading = ref(true)
const error = ref('')
const showClosed = ref(false)
const showSpecial = ref(false)
const healthFilter = ref('all')
const chartMetric = ref('size')
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)

const healthOptions = [
  { label: 'All', value: 'all' },
  { label: 'Healthy', value: 'healthy' },
  { label: 'Unhealthy', value: 'unhealthy' },
]
const chartMetrics = [
  { label: 'Size', value: 'size' },
  { label: 'Docs', value: 'docs' },
]

const stats = computed(() => {
  if (!data.value) return []
  return [
    { label: 'Total Indices', value: formatNumber(data.value.total_indices) },
    { label: 'Open / Closed', value: `${data.value.total_indices - data.value.closed_indices} / ${data.value.closed_indices}` },
    { label: 'Total Docs', value: formatNumber(data.value.docs_count) },
    { label: 'Total Store Size', value: formatBytes(data.value.size_in_bytes) },
    { label: 'Active Shards', value: formatNumber(data.value.active_shards) },
    { label: 'Unassigned Shards', value: formatNumber(data.value.unassigned_shards) },
  ]
})

const filteredIndices = computed(() => {
  let indices = data.value?.indices || []
  if (!showClosed.value) indices = indices.filter((i) => !i.closed)
  if (!showSpecial.value) indices = indices.filter((i) => !i.special)
  if (healthFilter.value === 'healthy') indices = indices.filter((i) => !i.unhealthy)
  if (healthFilter.value === 'unhealthy') indices = indices.filter((i) => i.unhealthy)
  return indices
})

const chartData = computed(() => {
  const top = [...filteredIndices.value]
    .sort((a, b) => chartMetric.value === 'docs'
      ? (b.doc_count || 0) - (a.doc_count || 0)
      : (b.total_size_in_bytes || 0) - (a.total_size_in_bytes || 0))
    .slice(0, 10)
  return {
    labels: top.map((i) => i.name),
    datasets: [{
      label: chartMetric.value === 'docs' ? 'Docs' : 'Total size',
      data: top.map((i) => chartMetric.value === 'docs' ? (i.doc_count || 0) : (i.total_size_in_bytes || 0)),
      backgroundColor: tokens.value.series.replica,
      borderRadius: 4,
    }],
  }
})

const columns: CerebroColumnDef<OverviewIndex>[] = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'status', header: 'Status', accessorFn: (r) => r.closed ? 'closed' : r.unhealthy ? 'unhealthy' : 'open' },
  { id: 'shards', header: 'Shards', accessorFn: (r) => r.num_shards },
  { id: 'doc_count', header: 'Docs', accessorFn: (r) => r.doc_count ?? 0 },
  { id: 'deleted_docs', header: 'Deleted', accessorFn: (r) => r.deleted_docs ?? 0 },
  { id: 'size_in_bytes', header: 'Primary Size', accessorFn: (r) => r.size_in_bytes ?? 0 },
  { id: 'total_size_in_bytes', header: 'Total Size', accessorFn: (r) => r.total_size_in_bytes ?? 0 },
  { id: 'aliases', header: 'Aliases', accessorFn: (r) => (r.aliases || []).join(', ') },
  { id: 'actions', header: 'Actions', sortable: false },
]

function indexActions(index: OverviewIndex) {
  return [
    [
      { label: 'Show settings', onSelect: () => inspect('Settings', '/commons/get_index_settings', index.name) },
      { label: 'Show mappings', onSelect: () => inspect('Mappings', '/commons/get_index_mapping', index.name) },
      { label: 'Show stats', onSelect: () => inspect('Stats', '/commons/get_index_stats', index.name) },
      { label: 'Edit index settings', onSelect: () => navigateTo(`/index-settings?index=${encodeURIComponent(index.name)}`) },
    ],
    [
      { label: index.closed ? 'Open' : 'Close', onSelect: () => runIndexAction(index.closed ? '/overview/open_indices' : '/overview/close_indices', index.name) },
      { label: 'Force merge', onSelect: () => runIndexAction('/overview/force_merge', index.name) },
      { label: 'Refresh', onSelect: () => runIndexAction('/overview/refresh_indices', index.name) },
      { label: 'Flush', onSelect: () => runIndexAction('/overview/flush_indices', index.name) },
      { label: 'Clear cache', onSelect: () => runIndexAction('/overview/clear_indices_cache', index.name) },
    ],
    [
      { label: 'Delete', color: 'error', onSelect: () => runIndexAction('/overview/delete_indices', index.name) },
    ],
  ]
}

async function inspect(title: string, url: string, index: string) {
  const resp = await apiCall('POST', url, { host: clusterHost.value, index })
  inspectorTitle.value = `${title}: ${index}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

async function runIndexAction(url: string, index: string) {
  await apiCall('POST', url, { host: clusterHost.value, indices: index })
  await load()
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/overview', { host: clusterHost.value })
  if (resp?.status === 200) {
    data.value = resp.body
    error.value = ''
  } else if (resp?.status === 401 || resp?.status === 403) {
    error.value = `Unauthorized (${resp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load indices (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load indices (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

onMounted(load)
</script>
