<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Snapshots</h4>
      <div class="flex flex-wrap items-center gap-2">
        <USelect v-model="selectedRepo" :items="repos" size="sm" class="w-48" placeholder="Select repository" @update:model-value="loadSnapshots" />
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="loadSnapshots(selectedRepo)">
          Refresh
        </UButton>
        <UButton size="sm" icon="i-lucide-camera" :disabled="!selectedRepo" @click="openCreate">Create Snapshot</UButton>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="snapshots-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>
    <UCard v-else-if="!repos.length" class="mt-3" data-testid="snapshots-empty">
      <p class="mb-3 text-sm text-neutral-500">No snapshot repositories are configured.</p>
      <UButton to="/repository" size="sm" color="primary" variant="outline">Configure Repository</UButton>
    </UCard>

    <div v-else>
      <CerebroTable
        :data="filteredSnapshots"
        :columns="columns"
        :default-sort="[{ id: 'start_time_in_millis', desc: true }]"
        filter-placeholder="Filter snapshots..."
        empty-text="No snapshots in this repository."
      >
        <template #filters>
          <USelect v-model="stateFilter" :items="stateOptions" size="sm" class="w-36" />
        </template>
        <template #state-cell="{ row }">
          <UBadge :color="stateColor(row.original.state)" variant="soft">{{ row.original.state }}</UBadge>
        </template>
        <template #start_time_in_millis-cell="{ row }">
          {{ formatTimestamp(row.original.start_time_in_millis) }}
        </template>
        <template #duration_in_millis-cell="{ row }">
          {{ formatDuration(row.original.duration_in_millis) }}
        </template>
        <template #indices_count-cell="{ row }">
          <span :title="(row.original.indices || []).join(', ')">{{ (row.original.indices || []).length }}</span>
        </template>
        <template #failed_shards-cell="{ row }">
          <div class="flex items-center gap-1">
            <span>{{ row.original.shards?.successful ?? 0 }}/{{ row.original.shards?.total ?? 0 }}</span>
            <UBadge v-if="(row.original.shards?.failed ?? 0) > 0 || row.original.failures?.length" color="error" variant="soft">
              {{ row.original.shards?.failed ?? row.original.failures?.length }} failed
            </UBadge>
          </div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex items-center gap-1">
            <UButton size="xs" color="primary" variant="outline" @click="openRestore(row.original)">Restore</UButton>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-search" @click="openInspect(row.original)" />
            <UButton size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="askDelete(row.original)" />
          </div>
        </template>
      </CerebroTable>
    </div>

    <UModal v-model:open="createOpen" title="Create Snapshot">
      <template #header>
        <h3 class="text-base font-semibold">Create Snapshot</h3>
      </template>
      <template #body>
        <div class="space-y-3">
          <UFormField label="Snapshot name">
            <UInput v-model="createForm.snapshot" class="w-full" data-testid="create-snapshot-name" />
          </UFormField>
          <UCheckbox v-model="createForm.showSystem" label="Show system indices" />
          <div class="max-h-48 overflow-y-auto rounded-lg border border-neutral-200 p-2 dark:border-neutral-800">
            <UCheckbox
              v-for="index in createIndices"
              :key="index"
              :model-value="createForm.indices.includes(index)"
              :label="index"
              @update:model-value="toggleCreateIndex(index, $event)"
            />
          </div>
          <UCheckbox v-model="createForm.ignoreUnavailable" label="Ignore unavailable indices" />
          <UCheckbox v-model="createForm.includeGlobalState" label="Include global cluster state" />
          <UAlert v-if="createError" color="error" variant="soft" :description="createError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="createOpen = false">Cancel</UButton>
          <UButton :loading="creating" data-testid="create-snapshot-submit" @click="createSnapshot">Create</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="inspectOpen" :title="`Snapshot ${inspectSnapshot?.snapshot || ''}`">
      <template #header>
        <h3 class="text-base font-semibold">Snapshot {{ inspectSnapshot?.snapshot }}</h3>
      </template>
      <template #body>
        <div v-if="inspectSnapshot" class="space-y-3">
          <pre class="max-h-64 overflow-auto rounded-lg bg-neutral-100 p-3 text-xs dark:bg-neutral-900"><code>{{ JSON.stringify(inspectSnapshot, null, 2) }}</code></pre>
          <div v-if="inspectSnapshot.failures?.length">
            <h4 class="mb-2 text-sm font-semibold text-error-500">Shard Failures</h4>
            <table class="w-full text-xs">
              <thead>
                <tr class="text-left text-neutral-500">
                  <th>Index</th><th>Shard</th><th>Node</th><th>Status</th><th>Reason</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(failure, i) in inspectSnapshot.failures" :key="i">
                  <td>{{ failure.index }}</td>
                  <td>{{ failure.shard_id }}</td>
                  <td>{{ failure.node_id }}</td>
                  <td>{{ failure.status }}</td>
                  <td>{{ failure.reason }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </UModal>

    <SnapshotRestoreModal
      v-model:open="restoreOpen"
      :snapshot="restoreSnapshot"
      :cluster-indices="clusterIndices"
      :repository="selectedRepo"
      @restored="load"
    />

    <ConfirmActionModal
      v-model:open="confirmOpen"
      title="Delete snapshot"
      :description="`Delete snapshot ${pendingDelete?.snapshot || ''}? This cannot be undone.`"
      confirm-label="Delete"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CerebroTable from '../../components/CerebroTable.vue'
import ConfirmActionModal from '../../components/ConfirmActionModal.vue'
import SnapshotRestoreModal from '../../components/snapshots/SnapshotRestoreModal.vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'
import type { SnapshotEntry } from '../../types/cerebro'
import { formatDuration, formatTimestamp } from '../../utils/format'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const repos = ref<string[]>([])
const selectedRepo = ref('')
const snapshots = ref<SnapshotEntry[]>([])
const clusterIndices = ref<string[]>([])
const loading = ref(true)
const error = ref('')
const stateFilter = ref('all')
const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')
const createForm = ref({ snapshot: '', indices: [] as string[], showSystem: false, ignoreUnavailable: true, includeGlobalState: false })
const inspectOpen = ref(false)
const inspectSnapshot = ref<SnapshotEntry | null>(null)
const restoreOpen = ref(false)
const restoreSnapshot = ref<SnapshotEntry | null>(null)
const confirmOpen = ref(false)
const pendingDelete = ref<SnapshotEntry | null>(null)

const stateOptions = [
  { label: 'All', value: 'all' },
  { label: 'SUCCESS', value: 'SUCCESS' },
  { label: 'PARTIAL', value: 'PARTIAL' },
  { label: 'FAILED', value: 'FAILED' },
  { label: 'IN_PROGRESS', value: 'IN_PROGRESS' },
]

const filteredSnapshots = computed(() => stateFilter.value === 'all'
  ? snapshots.value
  : snapshots.value.filter((s) => s.state === stateFilter.value))

const createIndices = computed(() => clusterIndices.value.filter((i) => createForm.value.showSystem || !i.startsWith('.')))

const columns: CerebroColumnDef<SnapshotEntry>[] = [
  { id: 'snapshot', header: 'Snapshot', accessorKey: 'snapshot' },
  { id: 'state', header: 'State', accessorKey: 'state' },
  { id: 'start_time_in_millis', header: 'Started', accessorFn: (s) => s.start_time_in_millis ?? 0 },
  { id: 'duration_in_millis', header: 'Duration', accessorFn: (s) => s.duration_in_millis ?? 0 },
  { id: 'indices_count', header: 'Indices', accessorFn: (s) => (s.indices || []).length },
  { id: 'failed_shards', header: 'Shards', accessorFn: (s) => s.shards?.failed ?? 0 },
  { id: 'actions', header: 'Actions', sortable: false },
]

function stateColor(state: string): string {
  switch (state) {
    case 'SUCCESS': return 'success'
    case 'IN_PROGRESS': return 'info'
    case 'PARTIAL': return 'warning'
    case 'FAILED':
    case 'INCOMPATIBLE': return 'error'
    default: return 'neutral'
  }
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/snapshots', { host: clusterHost.value })
  if (resp?.status === 200) {
    repos.value = resp.body?.repositories || []
    clusterIndices.value = (resp.body?.indices || []).map((i: any) => i.name)
    error.value = ''
    if (repos.value.length) {
      selectedRepo.value = selectedRepo.value || repos.value[0]!
      await loadSnapshots(selectedRepo.value)
    }
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load snapshots (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load snapshots (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

async function loadSnapshots(repo: string) {
  if (!repo) return
  loading.value = true
  const resp = await apiCall('POST', '/snapshots/load', { host: clusterHost.value, repository: repo })
  if (resp?.status === 200) {
    snapshots.value = resp.body || []
    error.value = ''
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load snapshots for ${repo} (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load snapshots for ${repo} (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

function openCreate() {
  createForm.value = { snapshot: '', indices: [], showSystem: false, ignoreUnavailable: true, includeGlobalState: false }
  createError.value = ''
  createOpen.value = true
}

function toggleCreateIndex(index: string, checked: boolean) {
  if (checked) {
    if (!createForm.value.indices.includes(index)) createForm.value.indices.push(index)
  } else {
    createForm.value.indices = createForm.value.indices.filter((i) => i !== index)
  }
}

async function createSnapshot() {
  creating.value = true
  createError.value = ''
  const resp = await apiCall('POST', '/snapshots/create', {
    host: clusterHost.value,
    repository: selectedRepo.value,
    snapshot: createForm.value.snapshot,
    indices: createForm.value.indices,
    ignoreUnavailable: createForm.value.ignoreUnavailable,
    includeGlobalState: createForm.value.includeGlobalState,
  }, { suppressErrorAlert: true })
  creating.value = false
  if (resp?.status === 200) {
    createOpen.value = false
    await loadSnapshots(selectedRepo.value)
  } else {
    createError.value = extractEnvelopeError(resp?.body) || `Failed to create snapshot (${resp?.status || 'network error'}).`
  }
}

function openInspect(snapshot: SnapshotEntry) {
  inspectSnapshot.value = snapshot
  inspectOpen.value = true
}

function openRestore(snapshot: SnapshotEntry) {
  restoreSnapshot.value = snapshot
  restoreOpen.value = true
}

function askDelete(snapshot: SnapshotEntry) {
  pendingDelete.value = snapshot
  confirmOpen.value = true
}

async function confirmDelete() {
  confirmOpen.value = false
  if (!pendingDelete.value) return
  await apiCall('POST', '/snapshots/delete', {
    host: clusterHost.value,
    repository: selectedRepo.value,
    snapshot: pendingDelete.value.snapshot,
  })
  pendingDelete.value = null
  await loadSnapshots(selectedRepo.value)
}

onMounted(load)
</script>
