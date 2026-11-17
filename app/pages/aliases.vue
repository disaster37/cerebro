<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Aliases</h4>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Refresh
        </UButton>
        <UButton size="sm" icon="i-lucide-plus" @click="openAdd">Add Alias</UButton>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="aliases-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <CerebroTable
      v-else
      :data="filteredAliases"
      :columns="columns"
      :default-sort="[{ id: 'alias', desc: false }]"
      filter-placeholder="Filter aliases..."
      empty-text="No aliases found."
    >
      <template #filters>
        <UInput v-model="aliasFilter" size="sm" icon="i-lucide-search" placeholder="Alias..." class="w-40" />
        <UInput v-model="indexFilter" size="sm" icon="i-lucide-search" placeholder="Index..." class="w-40" />
      </template>
      <template #filter-cell="{ row }">
        <code class="text-xs">{{ row.original.filter }}</code>
      </template>
      <template #actions-cell="{ row }">
        <UButton size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="removeAlias(row.original)">
          Remove
        </UButton>
      </template>
    </CerebroTable>

    <UModal v-model:open="addOpen" title="Add Alias">
      <template #header>
        <h3 class="text-base font-semibold">Add Alias</h3>
      </template>
      <template #body>
        <div class="space-y-3">
          <UFormField label="Alias">
            <UInput v-model="form.alias" class="w-full" data-testid="alias-name" />
          </UFormField>
          <UFormField label="Index">
            <UInput v-model="form.index" class="w-full" data-testid="alias-index" />
          </UFormField>
          <UAlert v-if="formError" color="error" variant="soft" :description="formError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="addOpen = false">Cancel</UButton>
          <UButton :loading="saving" data-testid="alias-submit" @click="addAlias">Add</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CerebroTable from '../components/CerebroTable.vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'
import type { CerebroColumnDef } from '../composables/useSortableColumns'

interface AliasEntry {
  alias: string
  index: string
  filter: any
  search_routing: any
  index_routing: any
}

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const aliases = ref<AliasEntry[]>([])
const loading = ref(true)
const error = ref('')
const aliasFilter = ref('')
const indexFilter = ref('')
const addOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const form = ref({ alias: '', index: '' })

const filteredAliases = computed(() => aliases.value.filter((a) => {
  if (aliasFilter.value && !a.alias.toLowerCase().includes(aliasFilter.value.toLowerCase())) return false
  if (indexFilter.value && !a.index.toLowerCase().includes(indexFilter.value.toLowerCase())) return false
  return true
}))

const columns: CerebroColumnDef<AliasEntry>[] = [
  { id: 'alias', header: 'Alias', accessorKey: 'alias' },
  { id: 'index', header: 'Index', accessorKey: 'index' },
  { id: 'filter', header: 'Filter', accessorFn: (a) => JSON.stringify(a.filter ?? null) },
  { id: 'search_routing', header: 'Search Routing', accessorFn: (a) => String(a.search_routing ?? '') },
  { id: 'index_routing', header: 'Index Routing', accessorFn: (a) => String(a.index_routing ?? '') },
  { id: 'actions', header: 'Actions', sortable: false },
]

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/aliases/get_aliases', { host: clusterHost.value })
  if (resp?.status === 200) {
    aliases.value = resp.body || []
    error.value = ''
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load aliases (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load aliases (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

function openAdd() {
  form.value = { alias: '', index: '' }
  formError.value = ''
  addOpen.value = true
}

async function addAlias() {
  saving.value = true
  formError.value = ''
  const resp = await apiCall('POST', '/aliases/update_aliases', {
    host: clusterHost.value,
    changes: [{ add: { index: form.value.index, alias: form.value.alias } }],
  }, { suppressErrorAlert: true })
  saving.value = false
  if (resp?.status === 200) {
    addOpen.value = false
    await load()
  } else {
    formError.value = extractEnvelopeError(resp?.body) || `Failed to add alias (${resp?.status || 'network error'}).`
  }
}

async function removeAlias(alias: AliasEntry) {
  await apiCall('POST', '/aliases/update_aliases', {
    host: clusterHost.value,
    changes: [{ remove: { index: alias.index, alias: alias.alias } }],
  })
  await load()
}

onMounted(load)
</script>
