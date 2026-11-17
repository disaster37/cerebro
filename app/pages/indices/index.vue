<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Indexes <small class="text-neutral-500">{{ clusterName }}</small></h4>
      <div class="flex items-center gap-2">
        <UButton
          size="sm"
          icon="i-lucide-layers"
          data-testid="open-create-datastream"
          @click="openCreateDataStream"
        >
          Create Data Stream
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Refresh
        </UButton>
      </div>
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

      <UTabs v-model="activeView" :items="tabs" class="mb-4" />

      <div v-if="activeView === 'indices'">
        <div
          v-if="selectedDataStream"
          class="mb-3 flex flex-wrap items-center gap-2 rounded-lg bg-primary-50 px-3 py-2 text-sm dark:bg-primary-950/40"
          data-testid="datastream-filter-chip"
        >
          <span>Data stream: <strong>{{ selectedDataStream }}</strong> ({{ backingCount }} backing indices)</span>
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="selectedDataStream = ''">Clear</UButton>
        </div>

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
            <UCheckbox v-model="showBacking" label="Show backing (.ds)" />
            <USelect v-model="healthFilter" :items="healthOptions" size="sm" class="w-32" />
            <USelect v-model="dataStreamFilter" :items="dataStreamOptions" size="sm" class="w-44" />
          </template>
          <template #name-cell="{ row }">
            <div class="flex flex-wrap items-center gap-1">
              <span>{{ row.original.name }}</span>
              <UBadge
                v-if="row.original.data_stream"
                color="primary"
                variant="soft"
                class="cursor-pointer"
                @click="viewBackingIndices({ name: row.original.data_stream })"
              >
                DS: {{ row.original.data_stream }}
              </UBadge>
              <UBadge v-if="row.original.is_write_index" color="info" variant="soft">write</UBadge>
            </div>
          </template>
          <template #status-cell="{ row }">
            <UBadge v-if="row.original.closed" color="neutral" variant="soft">closed</UBadge>
            <UBadge v-else-if="row.original.status === 'red'" color="error" variant="soft">red</UBadge>
            <UBadge v-else-if="row.original.status === 'yellow'" color="warning" variant="soft">yellow</UBadge>
            <UBadge v-else color="success" variant="soft">green</UBadge>
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

      <div v-else>
        <CerebroTable
          :data="dataStreams"
          :columns="dataStreamColumns"
          :default-sort="[{ id: 'name', desc: false }]"
          filter-placeholder="Filter data streams..."
          empty-text="No data streams found."
        >
          <template #status-cell="{ row }">
            <UBadge :color="dataStreamStatusColor(row.original.status)" variant="soft">{{ row.original.status }}</UBadge>
          </template>
          <template #backing-cell="{ row }">
            {{ (row.original.backing_indices || []).length }}
          </template>
          <template #doc_count-cell="{ row }">
            {{ formatNumber(row.original.doc_count) }}
          </template>
          <template #total_size_in_bytes-cell="{ row }">
            {{ formatBytes(row.original.total_size_in_bytes) }}
          </template>
          <template #actions-cell="{ row }">
            <div class="flex flex-wrap items-center gap-1">
              <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-list-tree" @click="viewBackingIndices(row.original)">
                View Backing Indices
              </UButton>
              <UButton size="xs" color="primary" variant="outline" icon="i-lucide-rotate-cw" @click="rolloverDataStream(row.original.name)">
                Rollover
              </UButton>
              <UButton size="xs" color="error" variant="outline" icon="i-lucide-trash-2" @click="confirmDeleteDataStream(row.original.name)">
                Delete
              </UButton>
            </div>
          </template>
        </CerebroTable>
      </div>
    </div>

    <JsonInspectorModal v-model:open="inspectorOpen" :title="inspectorTitle" :data="inspectorData" />

    <ConfirmActionModal
      v-model:open="confirmOpen"
      :title="confirmTitle"
      :description="confirmDescription"
      :confirm-label="confirmLabel"
      @confirm="runConfirmed"
    />

    <UModal v-model:open="createOpen" title="Create Data Stream">
      <template #header>
        <h3 class="text-base font-semibold">Create Data Stream</h3>
      </template>
      <template #body>
        <div class="space-y-3">
          <UFormField label="Data stream name">
            <UInput v-model="createForm.name" class="w-full" placeholder="logs-app" data-testid="datastream-name" />
          </UFormField>
          <UCheckbox v-model="createForm.createTemplate" label="Auto-create composable index template" />
          <template v-if="createForm.createTemplate">
            <UFormField label="Template name">
              <UInput v-model="createForm.templateName" class="w-full" :placeholder="`${createForm.name || 'logs-app'}-ds-template`" />
            </UFormField>
            <UFormField label="Index pattern">
              <UInput v-model="createForm.indexPattern" class="w-full" :placeholder="`${createForm.name || 'logs-app'}*`" />
            </UFormField>
            <div class="grid grid-cols-3 gap-2">
              <UFormField label="Priority">
                <UInput v-model.number="createForm.priority" type="number" class="w-full" />
              </UFormField>
              <UFormField label="Primary shards">
                <UInput v-model.number="createForm.numberOfShards" type="number" class="w-full" />
              </UFormField>
              <UFormField label="Replica shards">
                <UInput v-model.number="createForm.numberOfReplicas" type="number" class="w-full" />
              </UFormField>
            </div>
          </template>
          <UAlert v-if="createError" color="error" variant="soft" :description="createError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="createOpen = false">Cancel</UButton>
          <UButton :loading="creating" data-testid="datastream-submit" @click="createDataStream">Create</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Bar } from 'vue-chartjs'
import CerebroTable from '../../components/CerebroTable.vue'
import ConfirmActionModal from '../../components/ConfirmActionModal.vue'
import JsonInspectorModal from '../../components/JsonInspectorModal.vue'
import { extractEnvelopeError, pushAlert, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import { useChartTheme } from '../../composables/useChartTheme'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'
import type { DataStreamSummary, OverviewData, OverviewIndex } from '../../types/cerebro'
import { formatBytes, formatNumber } from '../../utils/format'

const { clusterHost, clusterName } = useCluster()
const { apiCall } = useAPI()
const { tokens, barOptions } = useChartTheme()
const colorMode = useColorMode()

const data = ref<OverviewData | null>(null)
const dataStreams = ref<DataStreamSummary[]>([])
const loading = ref(true)
const error = ref('')
const showClosed = ref(false)
const showSpecial = ref(false)
const showBacking = ref(true)
const healthFilter = ref('all')
const dataStreamFilter = ref('all')
const selectedDataStream = ref('')
const activeView = ref('indices')
const chartMetric = ref('size')
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)
const confirmOpen = ref(false)
const confirmTitle = ref('')
const confirmDescription = ref('')
const confirmLabel = ref('Confirm')
const pendingAction = ref<null | (() => Promise<void>)>(null)
const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')
const createForm = ref({
  name: '',
  createTemplate: true,
  templateName: '',
  indexPattern: '',
  priority: 200,
  numberOfShards: 1,
  numberOfReplicas: 1,
})

const healthOptions = [
  { label: 'All', value: 'all' },
  { label: 'Healthy', value: 'healthy' },
  { label: 'Unhealthy', value: 'unhealthy' },
]
const chartMetrics = [
  { label: 'Size', value: 'size' },
  { label: 'Docs', value: 'docs' },
]

const tabs = computed(() => [
  { label: `Indices (${filteredIndices.value.length})`, value: 'indices' },
  { label: `Data Streams (${dataStreams.value.length})`, value: 'datastreams' },
])

const dataStreamOptions = computed(() => [
  { label: 'All data streams', value: 'all' },
  ...dataStreams.value.map((ds) => ({ label: ds.name, value: ds.name })),
])

const backingCount = computed(() => (data.value?.indices || [])
  .filter((i) => i.data_stream === selectedDataStream.value).length)

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
  if (!showBacking.value) indices = indices.filter((i) => !i.backing_index)
  if (healthFilter.value === 'healthy') indices = indices.filter((i) => !i.unhealthy)
  if (healthFilter.value === 'unhealthy') indices = indices.filter((i) => i.unhealthy)
  if (dataStreamFilter.value !== 'all') indices = indices.filter((i) => i.data_stream === dataStreamFilter.value)
  if (selectedDataStream.value) indices = indices.filter((i) => i.data_stream === selectedDataStream.value)
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
  { id: 'status', header: 'Status', accessorFn: (r) => r.closed ? 'closed' : r.status },
  { id: 'shards', header: 'Shards', accessorFn: (r) => r.num_shards },
  { id: 'doc_count', header: 'Docs', accessorFn: (r) => r.doc_count ?? 0 },
  { id: 'deleted_docs', header: 'Deleted', accessorFn: (r) => r.deleted_docs ?? 0 },
  { id: 'size_in_bytes', header: 'Primary Size', accessorFn: (r) => r.size_in_bytes ?? 0 },
  { id: 'total_size_in_bytes', header: 'Total Size', accessorFn: (r) => r.total_size_in_bytes ?? 0 },
  { id: 'aliases', header: 'Aliases', accessorFn: (r) => (r.aliases || []).join(', ') },
  { id: 'actions', header: 'Actions', sortable: false },
]

const dataStreamColumns: CerebroColumnDef<DataStreamSummary>[] = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'status', header: 'Status', accessorKey: 'status' },
  { id: 'generation', header: 'Generation', accessorFn: (d) => d.generation ?? 0 },
  { id: 'write_index', header: 'Write Index', accessorKey: 'write_index' },
  { id: 'backing', header: 'Backing Indices', accessorFn: (d) => (d.backing_indices || []).length },
  { id: 'template', header: 'Template', accessorKey: 'template' },
  { id: 'doc_count', header: 'Docs', accessorFn: (d) => d.doc_count ?? 0 },
  { id: 'total_size_in_bytes', header: 'Total Size', accessorFn: (d) => d.total_size_in_bytes ?? 0 },
  { id: 'actions', header: 'Actions', sortable: false },
]

function dataStreamStatusColor(status: string) {
  switch (status) {
    case 'green': return 'success'
    case 'yellow': return 'warning'
    case 'red': return 'error'
    default: return 'neutral'
  }
}

function indexActions(index: OverviewIndex) {
  const closed = index.closed
  const groups: any[] = [
    [
      { label: 'Show settings', onSelect: () => inspect('Settings', '/commons/get_index_settings', index.name) },
      { label: 'Show mappings', onSelect: () => inspect('Mappings', '/commons/get_index_mapping', index.name) },
      { label: 'Show stats', onSelect: () => inspect('Stats', '/commons/get_index_stats', index.name) },
      { label: 'Edit index settings', onSelect: () => navigateTo(`/index-settings?index=${encodeURIComponent(index.name)}`) },
    ],
  ]
  if (index.data_stream) {
    groups.push([
      { label: 'View data stream backing indices', onSelect: () => viewBackingIndices({ name: index.data_stream! }) },
      { label: 'Rollover data stream', onSelect: () => rolloverDataStream(index.data_stream!) },
    ])
  }
  groups.push([
    {
      label: closed ? 'Open' : 'Close',
      onSelect: () => closed
        ? runIndexAction('/overview/open_indices', index.name, 'Opened index')
        : confirmIndexAction('Close index?', 'Close', '/overview/close_indices', index.name, 'Closed index'),
    },
    {
      label: 'Force merge',
      disabled: closed,
      onSelect: () => confirmIndexAction('Force merge index?', 'Force merge', '/overview/force_merge', index.name, 'Force merged index'),
    },
    { label: 'Refresh', disabled: closed, onSelect: () => runIndexAction('/overview/refresh_indices', index.name, 'Refreshed index') },
    { label: 'Flush', disabled: closed, onSelect: () => runIndexAction('/overview/flush_indices', index.name, 'Flushed index') },
    { label: 'Clear cache', disabled: closed, onSelect: () => runIndexAction('/overview/clear_indices_cache', index.name, 'Cleared cache for') },
  ])
  groups.push([{
    label: index.is_write_index ? 'Delete (write index — delete data stream instead)' : 'Delete',
    color: 'error',
    disabled: Boolean(index.is_write_index),
    onSelect: () => confirmIndexAction('Delete index? This cannot be undone.', 'Delete', '/overview/delete_indices', index.name, 'Deleted index'),
  }])
  return groups
}

function reportAction(resp: any, successLabel: string, target: string) {
  if (resp?.status === 200) {
    const failed = resp?.body?._shards?.failed
    if (typeof failed === 'number' && failed > 0) {
      pushAlert(`${successLabel} ${target} with ${failed} failed shard(s).`, 'warning')
    } else {
      pushAlert(`${successLabel} ${target}`, 'success')
    }
    return
  }
  const detail = extractEnvelopeError(resp?.body)
  pushAlert(
    detail
      ? `${successLabel} ${target} failed (${resp?.status || 'network error'}): ${detail}`
      : `${successLabel} ${target} failed (${resp?.status || 'network error'}).`,
    'danger',
  )
}

async function inspect(title: string, url: string, index: string) {
  const resp = await apiCall('POST', url, { host: clusterHost.value, index })
  inspectorTitle.value = `${title}: ${index}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

async function runIndexAction(url: string, index: string, successLabel: string) {
  const resp = await apiCall('POST', url, { host: clusterHost.value, indices: index }, { suppressErrorAlert: true })
  reportAction(resp, successLabel, index)
  await load()
}

function confirmIndexAction(title: string, label: string, url: string, index: string, successLabel: string) {
  confirmTitle.value = title
  confirmDescription.value = `This will affect index ${index}.`
  confirmLabel.value = label
  pendingAction.value = async () => {
    await runIndexAction(url, index, successLabel)
  }
  confirmOpen.value = true
}

async function runConfirmed() {
  confirmOpen.value = false
  if (pendingAction.value) await pendingAction.value()
  pendingAction.value = null
}

function viewBackingIndices(ds: { name: string }) {
  selectedDataStream.value = ds.name
  dataStreamFilter.value = 'all'
  showBacking.value = true
  activeView.value = 'indices'
}

async function rolloverDataStream(name: string) {
  const resp = await apiCall('POST', '/data_streams/rollover', { host: clusterHost.value, name }, { suppressErrorAlert: true })
  reportAction(resp, 'Rolled over data stream', name)
  await load()
}

function confirmDeleteDataStream(name: string) {
  confirmTitle.value = 'Delete data stream?'
  confirmDescription.value = `Delete data stream ${name} and all of its backing indices? This cannot be undone.`
  confirmLabel.value = 'Delete'
  pendingAction.value = async () => {
    const resp = await apiCall('POST', '/data_streams/delete', { host: clusterHost.value, name }, { suppressErrorAlert: true })
    reportAction(resp, 'Deleted data stream', name)
    if (selectedDataStream.value === name) selectedDataStream.value = ''
    await load()
  }
  confirmOpen.value = true
}

function openCreateDataStream() {
  createForm.value = {
    name: '',
    createTemplate: true,
    templateName: '',
    indexPattern: '',
    priority: 200,
    numberOfShards: 1,
    numberOfReplicas: 1,
  }
  createError.value = ''
  createOpen.value = true
}

function validateDataStreamName(name: string): string {
  if (!name) return 'Data stream name is required.'
  if (name !== name.toLowerCase()) return 'Data stream name must be lowercase.'
  if (/^[-_+]/.test(name)) return "Data stream name cannot start with '-', '_' or '+'."
  if (name.startsWith('.ds-')) return "Data stream name cannot start with '.ds-'."
  if (/[\s*?/\\,<>|"#:]/.test(name)) return 'Data stream name cannot contain spaces, wildcards, slashes or reserved characters.'
  return ''
}

async function createDataStream() {
  createError.value = validateDataStreamName(createForm.value.name)
  if (createError.value) return
  creating.value = true
  const resp = await apiCall('POST', '/data_streams/create', {
    host: clusterHost.value,
    name: createForm.value.name,
    createTemplate: createForm.value.createTemplate,
    templateName: createForm.value.templateName || undefined,
    indexPattern: createForm.value.indexPattern || undefined,
    priority: createForm.value.priority,
    numberOfShards: createForm.value.numberOfShards,
    numberOfReplicas: createForm.value.numberOfReplicas,
  }, { suppressErrorAlert: true })
  creating.value = false
  if (resp?.status === 200) {
    const created = createForm.value.name
    createOpen.value = false
    pushAlert(`Created data stream ${created}`, 'success')
    await load()
    selectedDataStream.value = created
    activeView.value = 'indices'
  } else {
    createError.value = extractEnvelopeError(resp?.body) || `Failed to create data stream (${resp?.status || 'network error'}).`
  }
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const [overviewResp, streamsResp] = await Promise.all([
    apiCall('POST', '/overview', { host: clusterHost.value }, { suppressErrorAlert: true }),
    apiCall('POST', '/data_streams', { host: clusterHost.value }, { suppressErrorAlert: true }),
  ])
  if (overviewResp?.status === 200) {
    data.value = overviewResp.body
    error.value = ''
  } else if (overviewResp?.status === 401 || overviewResp?.status === 403) {
    error.value = `Unauthorized (${overviewResp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(overviewResp?.body)
    error.value = detail
      ? `Failed to load indices (${overviewResp?.status || 'network error'}): ${detail}`
      : `Failed to load indices (${overviewResp?.status || 'network error'}).`
  }
  // Prefer the overview's data_streams (which carry aggregated doc/size stats
  // from backing indices); fall back to the raw /data_streams list when the
  // overview request failed.
  dataStreams.value = data.value?.data_streams
    || (streamsResp?.status === 200 && Array.isArray(streamsResp.body) ? streamsResp.body : [])
  loading.value = false
}

onMounted(load)
</script>
