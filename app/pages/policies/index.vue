<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <div>
        <h4 class="text-lg font-semibold">{{ title }}</h4>
        <div class="mt-1 flex items-center gap-2">
          <UBadge v-if="indexPayload" :color="indexPayload.operationMode === 'RUNNING' ? 'success' : 'warning'" variant="soft">
            {{ indexPayload.featureName }} {{ indexPayload.operationMode }}
          </UBadge>
          <UButton
            v-if="indexPayload && !indexPayload.unsupported"
            size="xs"
            color="neutral"
            variant="outline"
            :loading="runnerLoading"
            @click="toggleRunner"
          >
            {{ indexPayload.operationMode === 'RUNNING' ? 'Stop runner' : 'Start runner' }}
          </UButton>
        </div>
      </div>
      <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
        Refresh
      </UButton>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="policies-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else>
      <UAlert
        v-if="indexPayload?.unsupported"
        color="warning"
        variant="soft"
        class="mb-3"
        :description="indexPayload.warning"
        data-testid="policies-unsupported"
      />

      <UTabs v-model="activeTab" :items="tabs" class="mb-4">
        <template #badge-index>
          <UBadge v-if="failedIndices.length" color="error" variant="soft" class="ml-1">{{ failedIndices.length }}</UBadge>
        </template>
        <template #badge-snapshot>
          <UBadge v-if="failedSnapshotPolicies.length" color="error" variant="soft" class="ml-1">{{ failedSnapshotPolicies.length }}</UBadge>
        </template>
      </UTabs>

      <div v-if="activeTab === 'index'">
        <div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <UCard class="text-center">
            <p class="text-2xl font-semibold">{{ indexPayload?.policies?.length || 0 }}</p>
            <p class="text-xs text-neutral-500">Total Policies</p>
          </UCard>
          <UCard class="text-center">
            <p class="text-2xl font-semibold">{{ managedIndices.length }}</p>
            <p class="text-xs text-neutral-500">Managed Indices</p>
          </UCard>
          <UCard class="cursor-pointer text-center" @click="failedOnly = true">
            <p class="text-2xl font-semibold text-error-500">{{ failedIndices.length }}</p>
            <p class="text-xs text-neutral-500">Failed Indices</p>
          </UCard>
        </div>

        <h5 class="mb-2 text-sm font-semibold">Managed Indices & Policy Failures</h5>
        <CerebroTable
          :data="filteredManagedIndices"
          :columns="managedColumns"
          :default-sort="[{ id: 'index', desc: false }]"
          filter-placeholder="Filter managed indices..."
          empty-text="No managed indices."
        >
          <template #filters>
            <USwitch v-model="failedOnly" label="Failed indices only" />
            <USelect v-model="policyFilter" :items="policyOptions" size="sm" class="w-40" />
          </template>
          <template #actions>
            <UButton size="xs" color="error" variant="soft" :disabled="!failedIndices.length" @click="retryAllFailed">
              Retry All Failed ({{ failedIndices.length }})
            </UButton>
            <UButton size="xs" color="neutral" variant="outline" :disabled="!selectedFailed.length" @click="retrySelected">
              Retry Selected
            </UButton>
          </template>
          <template #select-cell="{ row }">
            <UCheckbox
              v-if="row.original.failed"
              :model-value="selectedFailed.includes(row.original.index)"
              @update:model-value="toggleSelected(row.original.index, $event)"
            />
          </template>
          <template #status-cell="{ row }">
            <UBadge :color="row.original.failed ? 'error' : 'success'" variant="soft">
              {{ row.original.failed ? 'FAILED' : 'HEALTHY' }}
            </UBadge>
          </template>
          <template #reason-cell="{ row }">
            <span class="text-xs">{{ row.original.reason || '-' }}</span>
          </template>
          <template #actions-cell="{ row }">
            <div class="flex flex-wrap items-center gap-1">
              <UButton
                v-if="row.original.failed"
                size="xs"
                color="error"
                variant="outline"
                icon="i-lucide-rotate-ccw"
                @click="openRetry(row.original)"
              >
                {{ isOpenSearch ? 'Retry Policy' : 'Retry Step' }}
              </UButton>
              <UButton
                size="xs"
                color="neutral"
                variant="outline"
                icon="i-lucide-eye"
                @click="inspect(`Managed Index: ${row.original.index}`, row.original)"
              >
                Details
              </UButton>
            </div>
          </template>
        </CerebroTable>

        <h5 class="mb-2 mt-6 text-sm font-semibold">Defined Policies</h5>
        <CerebroTable
          :data="indexPayload?.policies || []"
          :columns="policyColumns"
          :default-sort="[{ id: 'id', desc: false }]"
          filter-placeholder="Filter policies..."
          empty-text="No lifecycle policies."
        >
          <template #phases-cell="{ row }">
            <div class="flex flex-wrap gap-1">
              <UBadge v-for="phase in row.original.phases" :key="phase" color="info" variant="soft">{{ phase }}</UBadge>
            </div>
          </template>
          <template #actions-cell="{ row }">
            <UButton
              size="xs"
              color="neutral"
              variant="outline"
              icon="i-lucide-file-code"
              @click="inspect(`Policy: ${row.original.id}`, row.original.raw_policy)"
            >
              View JSON
            </UButton>
          </template>
        </CerebroTable>
      </div>

      <div v-else>
        <UAlert
          v-if="snapshotUnsupported"
          color="warning"
          variant="soft"
          class="mb-3"
          :description="snapshotWarning"
        />
        <CerebroTable
          :data="filteredSnapshotPolicies"
          :columns="snapshotColumns"
          :default-sort="[{ id: 'id', desc: false }]"
          filter-placeholder="Filter snapshot policies..."
          empty-text="No snapshot lifecycle policies."
        >
          <template #filters>
            <USwitch v-model="snapshotFailedOnly" label="Failed only" />
          </template>
          <template #status-cell="{ row }">
            <div class="flex items-center gap-1">
              <UBadge :color="row.original.enabled ? 'success' : 'neutral'" variant="soft">
                {{ row.original.enabled ? 'Enabled' : 'Disabled' }}
              </UBadge>
              <UBadge color="info" variant="soft">{{ row.original.current_state }}</UBadge>
            </div>
          </template>
          <template #last_success-cell="{ row }">
            <div v-if="row.original.last_success_name">
              <div class="text-xs">{{ row.original.last_success_name }}</div>
              <div class="text-[11px] text-neutral-500">{{ formatTimestamp(row.original.last_success_time) }}</div>
            </div>
            <span v-else>-</span>
          </template>
          <template #last_failure-cell="{ row }">
            <div v-if="row.original.last_failure_message" class="space-y-1">
              <p class="max-w-xs text-xs text-error-500">{{ row.original.last_failure_message }}</p>
              <UButton size="xs" color="error" variant="ghost" @click="inspect('Snapshot policy failure', row.original)">Inspect Error</UButton>
            </div>
            <span v-else>-</span>
          </template>
          <template #taken-cell="{ row }">
            {{ row.original.snapshots_taken }} / {{ row.original.snapshots_failed }}
          </template>
          <template #actions-cell="{ row }">
            <div class="flex flex-wrap items-center gap-1">
              <template v-if="isOpenSearch">
                <UButton size="xs" color="success" variant="outline" icon="i-lucide-play" @click="snapshotAction(row.original.id, 'start')">Start</UButton>
                <UButton size="xs" color="warning" variant="outline" icon="i-lucide-square" @click="snapshotAction(row.original.id, 'stop')">Stop</UButton>
                <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="snapshotAction(row.original.id, 'restart')">Restart</UButton>
              </template>
              <UButton v-else size="xs" color="primary" variant="outline" icon="i-lucide-play" @click="snapshotAction(row.original.id, 'execute')">Run Now / Retry</UButton>
              <UButton
                size="xs"
                color="neutral"
                variant="outline"
                icon="i-lucide-file-code"
                @click="inspect(`Snapshot Policy: ${row.original.id}`, row.original.raw_policy)"
              >
                View JSON
              </UButton>
            </div>
          </template>
        </CerebroTable>
      </div>
    </div>

    <UModal v-model:open="retryOpen" title="Retry lifecycle policy">
      <template #header>
        <h3 class="text-base font-semibold">Retry {{ retryRow?.index }}</h3>
      </template>
      <template #body>
        <div v-if="retryRow" class="space-y-3">
          <template v-if="isOpenSearch">
            <UCheckbox v-model="retryFromState" label="Restart from specific state" />
            <USelect v-if="retryFromState" v-model="retryState" :items="retryRow.available_states || []" class="w-full" />
            <p v-else class="text-xs text-neutral-500">Retry from the failed action.</p>
          </template>
          <p v-else class="text-xs text-neutral-500">Retry the failed ILM step for this index.</p>
          <UAlert v-if="retryError" color="error" variant="soft" :description="retryError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="retryOpen = false">Cancel</UButton>
          <UButton :loading="retrying" @click="confirmRetry">Retry</UButton>
        </div>
      </template>
    </UModal>

    <JsonInspectorModal v-model:open="inspectorOpen" :title="inspectorTitle" :data="inspectorData" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CerebroTable from '../../components/CerebroTable.vue'
import JsonInspectorModal from '../../components/JsonInspectorModal.vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'
import type {
  IndexLifecyclePayload,
  IndexPolicySummary,
  ManagedIndexStatus,
  SnapshotPolicySummary,
} from '../../types/cerebro'
import { formatTimestamp } from '../../utils/format'

const { clusterHost, backendType } = useCluster()
const { apiCall } = useAPI()

const indexPayload = ref<IndexLifecyclePayload | null>(null)
const snapshotPolicies = ref<SnapshotPolicySummary[]>([])
const snapshotUnsupported = ref(false)
const snapshotWarning = ref('')
const loading = ref(true)
const error = ref('')
const activeTab = ref('index')
const failedOnly = ref(false)
const policyFilter = ref('all')
const selectedFailed = ref<string[]>([])
const snapshotFailedOnly = ref(false)
const runnerLoading = ref(false)
const retryOpen = ref(false)
const retryRow = ref<ManagedIndexStatus | null>(null)
const retryFromState = ref(false)
const retryState = ref('')
const retryError = ref('')
const retrying = ref(false)
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)

const isOpenSearch = computed(() => (indexPayload.value?.backendType || backendType.value) === 'opensearch')

const title = computed(() => isOpenSearch.value
  ? 'State & Snapshot Management (ISM & SM)'
  : 'Lifecycle Policies (ILM & SLM)')

const tabs = [
  { label: 'Index Policies', value: 'index' },
  { label: 'Snapshot Policies', value: 'snapshot' },
]

const managedIndices = computed(() => indexPayload.value?.managedIndices || [])
const failedIndices = computed(() => managedIndices.value.filter((i) => i.failed))
const failedSnapshotPolicies = computed(() => snapshotPolicies.value.filter((p) => Boolean(p.last_failure_message)))

const policyOptions = computed(() => [
  { label: 'All policies', value: 'all' },
  ...(indexPayload.value?.policies || []).map((p) => ({ label: p.id, value: p.id })),
])

const filteredManagedIndices = computed(() => managedIndices.value.filter((row) => {
  if (failedOnly.value && !row.failed) return false
  if (policyFilter.value !== 'all' && row.policy_id !== policyFilter.value) return false
  return true
}))

const filteredSnapshotPolicies = computed(() => snapshotFailedOnly.value
  ? failedSnapshotPolicies.value
  : snapshotPolicies.value)

const managedColumns: CerebroColumnDef<ManagedIndexStatus>[] = [
  { id: 'select', header: '', sortable: false },
  { id: 'index', header: 'Index', accessorKey: 'index' },
  { id: 'policy_id', header: 'Policy', accessorKey: 'policy_id' },
  { id: 'phase', header: 'Phase / State', accessorKey: 'phase' },
  { id: 'action', header: 'Action / Step', accessorFn: (r) => `${r.action || ''}${r.step ? ` / ${r.step}` : ''}` },
  { id: 'status', header: 'Status', accessorFn: (r) => r.failed ? 'FAILED' : 'HEALTHY' },
  { id: 'reason', header: 'Error / Status Detail', accessorFn: (r) => r.reason || '' },
  { id: 'actions', header: 'Actions', sortable: false },
]

const policyColumns: CerebroColumnDef<IndexPolicySummary>[] = [
  { id: 'id', header: 'Policy ID', accessorKey: 'id' },
  { id: 'version', header: 'Version', accessorFn: (p) => p.version ?? 0 },
  { id: 'phases', header: 'Phases / States', accessorFn: (p) => (p.phases || []).join(' -> ') },
  { id: 'managed_count', header: 'Managed Indices', accessorFn: (p) => p.managed_count ?? 0 },
  { id: 'failed_count', header: 'Failed Indices', accessorFn: (p) => p.failed_count ?? 0 },
  { id: 'modified_date', header: 'Last Modified', accessorKey: 'modified_date' },
  { id: 'actions', header: 'Actions', sortable: false },
]

const snapshotColumns: CerebroColumnDef<SnapshotPolicySummary>[] = [
  { id: 'id', header: 'Policy', accessorKey: 'id' },
  { id: 'repository', header: 'Repository', accessorKey: 'repository' },
  { id: 'schedule', header: 'Schedule', accessorKey: 'schedule' },
  { id: 'status', header: 'Status', accessorFn: (p) => `${p.enabled ? 'Enabled' : 'Disabled'} ${p.current_state}` },
  { id: 'last_success', header: 'Last Success', accessorFn: (p) => p.last_success_time ?? 0 },
  { id: 'last_failure', header: 'Last Failure', accessorFn: (p) => p.last_failure_time ?? 0 },
  { id: 'taken', header: 'Taken / Failed', accessorFn: (p) => p.snapshots_taken ?? 0 },
  { id: 'actions', header: 'Actions', sortable: false },
]

function toggleSelected(index: string, checked: boolean) {
  if (checked) {
    if (!selectedFailed.value.includes(index)) selectedFailed.value.push(index)
  } else {
    selectedFailed.value = selectedFailed.value.filter((i) => i !== index)
  }
}

function openRetry(row: ManagedIndexStatus) {
  retryRow.value = row
  retryFromState.value = false
  retryState.value = row.available_states?.[0] || ''
  retryError.value = ''
  retryOpen.value = true
}

async function retryIndices(indices: string[], state?: string) {
  const body: Record<string, any> = { host: clusterHost.value, indices }
  if (state) body.state = state
  const resp = await apiCall('POST', '/policies/retry_indices', body, { suppressErrorAlert: true })
  if (resp?.status === 200) {
    await load()
    return true
  }
  retryError.value = extractEnvelopeError(resp?.body) || `Retry failed (${resp?.status || 'network error'}).`
  return false
}

async function confirmRetry() {
  if (!retryRow.value) return
  retrying.value = true
  const state = isOpenSearch.value && retryFromState.value ? retryState.value : undefined
  const ok = await retryIndices([retryRow.value.index], state)
  retrying.value = false
  if (ok) retryOpen.value = false
}

async function retryAllFailed() {
  await retryIndices(failedIndices.value.map((i) => i.index))
}

async function retrySelected() {
  await retryIndices(selectedFailed.value)
  selectedFailed.value = []
}

async function toggleRunner() {
  if (!indexPayload.value) return
  runnerLoading.value = true
  await apiCall('POST', '/policies/toggle_engine', {
    host: clusterHost.value,
    start: indexPayload.value.operationMode !== 'RUNNING',
  })
  runnerLoading.value = false
  await load()
}

async function snapshotAction(policyId: string, action: string) {
  await apiCall('POST', '/policies/snapshot_action', {
    host: clusterHost.value,
    policyId,
    action,
  })
  await load()
}

function inspect(title: string, data: any) {
  inspectorTitle.value = title
  inspectorData.value = data
  inspectorOpen.value = true
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const [indexResp, snapshotResp] = await Promise.all([
    apiCall('POST', '/policies/index_lifecycle', { host: clusterHost.value }, { suppressErrorAlert: true }),
    apiCall('POST', '/policies/snapshot_lifecycle', { host: clusterHost.value }, { suppressErrorAlert: true }),
  ])

  if (indexResp?.status === 200) {
    indexPayload.value = indexResp.body
    error.value = ''
  } else {
    const detail = extractEnvelopeError(indexResp?.body)
    error.value = detail
      ? `Failed to load lifecycle policies (${indexResp?.status || 'network error'}): ${detail}`
      : `Failed to load lifecycle policies (${indexResp?.status || 'network error'}).`
  }

  if (snapshotResp?.status === 200) {
    const body = snapshotResp.body
    if (Array.isArray(body)) {
      snapshotPolicies.value = body
      snapshotUnsupported.value = false
    } else {
      snapshotPolicies.value = body?.policies || []
      snapshotUnsupported.value = Boolean(body?.unsupported)
      snapshotWarning.value = body?.warning || ''
    }
  }
  loading.value = false
}

onMounted(load)
</script>
