<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Repositories</h4>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-shield-check" :loading="verifyingAll" @click="verifyAll">
          Verify All
        </UButton>
        <UButton size="sm" icon="i-lucide-plus" @click="openCreate">Create Repository</UButton>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="repositories-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else>
      <CerebroTable
        :data="filteredRepos"
        :columns="columns"
        :default-sort="[{ id: 'name', desc: false }]"
        filter-placeholder="Filter repositories..."
        empty-text="No repositories configured."
      >
        <template #filters>
          <USelect v-model="typeFilter" :items="typeOptions" size="sm" class="w-32" />
        </template>
        <template #type-cell="{ row }">
          <UBadge color="info" variant="soft">{{ row.original.type }}</UBadge>
        </template>
        <template #location-cell="{ row }">
          <span class="text-xs">{{ locationOf(row.original) }}</span>
        </template>
        <template #status-cell="{ row }">
          <div v-if="checkMap[row.original.name] === 'checking'" class="flex items-center gap-1 text-xs">
            <UIcon name="i-lucide-loader-circle" class="size-3 animate-spin" /> Checking...
          </div>
          <div v-else-if="checkMap[row.original.name]" class="space-y-1">
            <UBadge :color="statusColor(row.original.name)" variant="soft">
              {{ statusLabel(row.original.name) }}
            </UBadge>
            <p v-if="checkMap[row.original.name]?.error" class="text-[11px] text-error-500">
              {{ checkMap[row.original.name]?.error }}
            </p>
          </div>
          <UBadge v-else color="neutral" variant="soft">Unchecked</UBadge>
        </template>
        <template #snapshot_count-cell="{ row }">
          {{ checkMap[row.original.name]?.snapshot_count ?? '-' }}
        </template>
        <template #size-cell="{ row }">
          <div v-if="checkMap[row.original.name]?.size_computed">
            {{ formatBytes(checkMap[row.original.name]?.size_in_bytes) }}
            <div class="text-[11px] text-neutral-500">logical: {{ formatBytes(checkMap[row.original.name]?.logical_size_in_bytes) }}</div>
          </div>
          <UButton
            v-else-if="checkMap[row.original.name]?.size_skipped_reason"
            size="xs"
            color="neutral"
            variant="ghost"
            @click="verify(row.original.name, true)"
          >
            Calculate Size
          </UButton>
          <span v-else>-</span>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex items-center gap-1">
            <UButton size="xs" color="neutral" variant="outline" :loading="checkMap[row.original.name] === 'checking'" @click="verify(row.original.name)">
              Verify
            </UButton>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-pencil" @click="openEdit(row.original)" />
            <UButton size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="askDelete(row.original)" />
          </div>
        </template>
      </CerebroTable>
    </div>

    <UModal v-model:open="formOpen" :title="editing ? 'Edit Repository' : 'Create Repository'">
      <template #header>
        <h3 class="text-base font-semibold">{{ editing ? 'Edit Repository' : 'Create Repository' }}</h3>
      </template>
      <template #body>
        <div class="space-y-3">
          <UFormField label="Name">
            <UInput v-model="form.name" class="w-full" :disabled="Boolean(editing)" data-testid="repository-name" />
          </UFormField>
          <UFormField label="Type">
            <USelect v-model="form.type" :items="repositoryTypes" class="w-full" data-testid="repository-type" />
          </UFormField>
          <UFormField label="Location / Bucket / URL">
            <UInput v-model="form.location" class="w-full" placeholder="/tmp/backups or my-bucket" data-testid="repository-location" />
          </UFormField>
          <UFormField label="Base path">
            <UInput v-model="form.basePath" class="w-full" placeholder="snapshots" />
          </UFormField>
          <div class="flex gap-4">
            <UCheckbox v-model="form.compress" label="Compress" />
            <UCheckbox v-model="form.readonly" label="Read only" />
          </div>
          <UFormField label="Additional settings (JSON)">
            <UTextarea v-model="form.extra" :rows="4" class="w-full font-mono" placeholder='{"max_snapshot_bytes_per_sec":"40mb"}' />
          </UFormField>
          <UAlert v-if="formError" color="error" variant="soft" :description="formError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="formOpen = false">Cancel</UButton>
          <UButton :loading="saving" data-testid="repository-submit" @click="save">Save</UButton>
        </div>
      </template>
    </UModal>

    <ConfirmActionModal
      v-model:open="confirmOpen"
      title="Delete repository"
      :description="`Delete repository ${pendingDelete || ''}? Snapshots stored in it will no longer be accessible.`"
      confirm-label="Delete"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CerebroTable from '../components/CerebroTable.vue'
import ConfirmActionModal from '../components/ConfirmActionModal.vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'
import type { CerebroColumnDef } from '../composables/useSortableColumns'
import type { RepositoryEntry, RepositoryVerifyStatus } from '../types/cerebro'
import { formatBytes } from '../utils/format'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const repos = ref<RepositoryEntry[]>([])
const loading = ref(true)
const error = ref('')
const typeFilter = ref('all')
const checkMap = ref<Record<string, RepositoryVerifyStatus | 'checking'>>({})
const verifyingAll = ref(false)
const formOpen = ref(false)
const editing = ref<RepositoryEntry | null>(null)
const saving = ref(false)
const formError = ref('')
const confirmOpen = ref(false)
const pendingDelete = ref('')

const repositoryTypes = ['fs', 's3', 'gcs', 'azure', 'url', 'hdfs']
const typeOptions = [{ label: 'All', value: 'all' }, ...repositoryTypes.map((t) => ({ label: t, value: t }))]

const form = ref({
  name: '',
  type: 'fs',
  location: '',
  basePath: '',
  compress: true,
  readonly: false,
  extra: '{}',
})

const filteredRepos = computed(() => typeFilter.value === 'all'
  ? repos.value
  : repos.value.filter((r) => r.type === typeFilter.value))

const columns: CerebroColumnDef<RepositoryEntry>[] = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'type', header: 'Type', accessorKey: 'type', class: 'whitespace-nowrap' },
  { id: 'location', header: 'Location / Settings', accessorFn: (r) => locationOf(r) },
  { id: 'status', header: 'Status', accessorFn: (r) => statusLabel(r.name), class: 'whitespace-nowrap' },
  { id: 'snapshot_count', header: 'Snapshots', accessorFn: (r) => checkMap.value[r.name]?.snapshot_count ?? -1, class: 'whitespace-nowrap' },
  { id: 'size', header: 'Size', accessorFn: (r) => checkMap.value[r.name]?.size_in_bytes ?? -1, class: 'whitespace-nowrap' },
  { id: 'actions', header: 'Actions', sortable: false, class: 'whitespace-nowrap' },
]

function locationOf(repo: RepositoryEntry): string {
  const settings = repo.settings || {}
  return settings.location || settings.bucket || settings.url || settings.container || JSON.stringify(settings)
}

function statusLabel(name: string): string {
  const status = checkMap.value[name]
  if (!status || status === 'checking') return 'Unchecked'
  if (status.status === 'ERROR') return 'Verification failed'
  if (status.status === 'PARTIAL_FAILURES') return `Snapshots failing (${status.failed_count})`
  if (status.status === 'IN_PROGRESS') return 'Verification in progress'
  return `Connected (${status.node_count} nodes)`
}

function statusColor(name: string): string {
  const status = checkMap.value[name]
  if (!status || status === 'checking') return 'neutral'
  if (status.status === 'ERROR') return 'error'
  if (status.status === 'PARTIAL_FAILURES') return 'warning'
  return 'success'
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/repositories', { host: clusterHost.value }, { suppressErrorAlert: true })
  if (resp?.status === 200) {
    repos.value = resp.body || []
    error.value = ''
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load repositories (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load repositories (${resp?.status || 'network error'}).`
  }
  loading.value = false
  // Populate status/snapshots/size immediately for small repository lists
  // instead of leaving every row "Unchecked" until a manual click.
  if (repos.value.length > 0 && repos.value.length <= 10) {
    void verifyAll()
  }
}

async function verify(name: string, forceFullSize = false) {
  checkMap.value[name] = 'checking'
  const resp = await apiCall('POST', '/repositories/verify', {
    host: clusterHost.value,
    name,
    forceFullSize,
  }, { suppressErrorAlert: true })
  if (resp?.status === 200) {
    checkMap.value[name] = resp.body
  } else {
    checkMap.value[name] = {
      name,
      verified: false,
      status: 'ERROR',
      node_count: 0,
      nodes: [],
      snapshot_count: 0,
      active_count: 0,
      failed_count: 0,
      size_computed: false,
      size_in_bytes: null,
      logical_size_in_bytes: null,
      error: extractEnvelopeError(resp?.body) || `Verification failed (${resp?.status || 'network error'})`,
    }
  }
}

async function verifyAll() {
  verifyingAll.value = true
  for (const repo of repos.value) {
    await verify(repo.name)
  }
  verifyingAll.value = false
}

function openCreate() {
  editing.value = null
  form.value = { name: '', type: 'fs', location: '', basePath: '', compress: true, readonly: false, extra: '{}' }
  formError.value = ''
  formOpen.value = true
}

function openEdit(repo: RepositoryEntry) {
  editing.value = repo
  const settings = repo.settings || {}
  form.value = {
    name: repo.name,
    type: repo.type,
    location: settings.location || settings.bucket || settings.url || '',
    basePath: settings.base_path || '',
    compress: settings.compress !== false,
    readonly: Boolean(settings.readonly),
    extra: JSON.stringify(settings, null, 2),
  }
  formError.value = ''
  formOpen.value = true
}

async function save() {
  formError.value = ''
  let extra: Record<string, any> = {}
  try {
    extra = form.value.extra ? JSON.parse(form.value.extra) : {}
  } catch {
    formError.value = 'Additional settings must be valid JSON.'
    return
  }
  const settings: Record<string, any> = { ...extra }
  if (form.value.location) {
    if (form.value.type === 's3' || form.value.type === 'gcs' || form.value.type === 'azure') settings.bucket = form.value.location
    else if (form.value.type === 'url') settings.url = form.value.location
    else settings.location = form.value.location
  }
  if (form.value.basePath) settings.base_path = form.value.basePath
  settings.compress = form.value.compress
  settings.readonly = form.value.readonly

  saving.value = true
  const resp = await apiCall('POST', '/repositories/create', {
    host: clusterHost.value,
    name: form.value.name,
    type: form.value.type,
    settings,
  }, { suppressErrorAlert: true })
  saving.value = false
  if (resp?.status === 200) {
    formOpen.value = false
    await load()
  } else {
    formError.value = extractEnvelopeError(resp?.body) || `Failed to save repository (${resp?.status || 'network error'}).`
  }
}

function askDelete(repo: RepositoryEntry) {
  pendingDelete.value = repo.name
  confirmOpen.value = true
}

async function confirmDelete() {
  confirmOpen.value = false
  if (!pendingDelete.value) return
  await apiCall('POST', '/repositories/delete', { host: clusterHost.value, name: pendingDelete.value })
  pendingDelete.value = ''
  await load()
}

onMounted(load)
</script>
