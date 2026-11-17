<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Templates</h4>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Refresh
        </UButton>
        <UButton size="sm" icon="i-lucide-plus" @click="openCreate">Create Template</UButton>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="templates-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <CerebroTable
      v-else
      :data="filteredTemplates"
      :columns="columns"
      :default-sort="[{ id: 'name', desc: false }]"
      :row-key="(t) => `${t.name}:${t.type}`"
      filter-placeholder="Filter templates..."
      empty-text="No templates found."
    >
      <template #filters>
        <USelect v-model="typeFilter" :items="typeOptions" size="sm" class="w-36" />
        <UInput v-model="patternFilter" size="sm" icon="i-lucide-search" placeholder="Pattern..." class="w-40" />
      </template>
      <template #type-cell="{ row }">
        <div class="flex flex-wrap items-center gap-1">
          <UBadge :color="row.original.type === 'composable' ? 'info' : 'neutral'" variant="soft">{{ row.original.type }}</UBadge>
          <UBadge v-if="row.original.data_stream" color="primary" variant="soft">data stream</UBadge>
        </div>
      </template>
      <template #patterns-cell="{ row }">
        <span class="text-xs">{{ (row.original.patterns || []).join(', ') }}</span>
      </template>
      <template #actions-cell="{ row }">
        <div class="flex items-center gap-1">
          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-file-code" @click="inspect(row.original)">View JSON</UButton>
          <UButton size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="del(row.original)" />
        </div>
      </template>
    </CerebroTable>

    <UModal v-model:open="createOpen" title="Create Template">
      <template #header>
        <h3 class="text-base font-semibold">Create Template</h3>
      </template>
      <template #body>
        <div class="space-y-3">
          <UFormField label="Name">
            <UInput v-model="form.name" class="w-full" data-testid="template-name" />
          </UFormField>
          <UFormField label="Type">
            <USelect v-model="form.type" :items="['composable', 'legacy']" class="w-full" />
          </UFormField>
          <UFormField label="Index patterns (comma-separated)">
            <UInput v-model="form.patterns" class="w-full" placeholder="logs-*, metrics-*" />
          </UFormField>
          <UFormField label="Template JSON">
            <UTextarea v-model="form.body" :rows="8" class="w-full font-mono" />
          </UFormField>
          <UAlert v-if="formError" color="error" variant="soft" :description="formError" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="createOpen = false">Cancel</UButton>
          <UButton :loading="saving" data-testid="template-submit" @click="create">Create</UButton>
        </div>
      </template>
    </UModal>

    <JsonInspectorModal v-model:open="inspectorOpen" :title="inspectorTitle" :data="inspectorData" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import CerebroTable from '../components/CerebroTable.vue'
import JsonInspectorModal from '../components/JsonInspectorModal.vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'
import type { CerebroColumnDef } from '../composables/useSortableColumns'
import type { TemplateEntry } from '../types/cerebro'

const COMPOSABLE_DEFAULT = '{\n  "template": {\n    "settings": {}\n  }\n}'
const LEGACY_DEFAULT = '{\n  "settings": {}\n}'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const templates = ref<TemplateEntry[]>([])
const loading = ref(true)
const error = ref('')
const typeFilter = ref('all')
const patternFilter = ref('')
const createOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const form = ref({ name: '', type: 'composable', patterns: '', body: COMPOSABLE_DEFAULT })
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)

const typeOptions = [
  { label: 'All', value: 'all' },
  { label: 'Composable', value: 'composable' },
  { label: 'Legacy', value: 'legacy' },
]

const filteredTemplates = computed(() => templates.value.filter((t) => {
  if (typeFilter.value !== 'all' && t.type !== typeFilter.value) return false
  if (patternFilter.value && !(t.patterns || []).some((p) => p.toLowerCase().includes(patternFilter.value.toLowerCase()))) return false
  return true
}))

const columns: CerebroColumnDef<TemplateEntry>[] = [
  { id: 'name', header: 'Name', accessorKey: 'name' },
  { id: 'type', header: 'Type', accessorKey: 'type' },
  { id: 'patterns', header: 'Patterns', accessorFn: (t) => (t.patterns || []).join(', ') },
  { id: 'order', header: 'Order', accessorFn: (t) => t.order ?? 0 },
  { id: 'version', header: 'Version', accessorFn: (t) => t.version ?? 0 },
  { id: 'actions', header: 'Actions', sortable: false },
]

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/templates', { host: clusterHost.value }, { suppressErrorAlert: true })
  if (resp?.status === 200) {
    templates.value = resp.body || []
    error.value = ''
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load templates (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load templates (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

function openCreate() {
  form.value = { name: '', type: 'composable', patterns: '', body: COMPOSABLE_DEFAULT }
  formError.value = ''
  createOpen.value = true
}

async function create() {
  formError.value = ''
  let body: any
  try {
    body = JSON.parse(form.value.body)
  } catch {
    formError.value = 'Template body must be valid JSON.'
    return
  }
  if (form.value.patterns) {
    body.index_patterns = form.value.patterns.split(',').map((p) => p.trim()).filter(Boolean)
  }
  saving.value = true
  const resp = await apiCall('POST', '/templates/create', {
    host: clusterHost.value,
    name: form.value.name,
    type: form.value.type,
    template: body,
  }, { suppressErrorAlert: true })
  saving.value = false
  if (resp?.status === 200) {
    createOpen.value = false
    await load()
  } else {
    formError.value = extractEnvelopeError(resp?.body) || `Failed to create template (${resp?.status || 'network error'}).`
  }
}

async function del(template: TemplateEntry) {
  await apiCall('POST', '/templates/delete', {
    host: clusterHost.value,
    name: template.name,
    type: template.type,
  })
  await load()
}

function inspect(template: TemplateEntry) {
  inspectorTitle.value = `Template: ${template.name}`
  inspectorData.value = template.template
  inspectorOpen.value = true
}

watch(() => form.value.type, (type, oldType) => {
  const oldDefault = oldType === 'legacy' ? LEGACY_DEFAULT : COMPOSABLE_DEFAULT
  if (form.value.body === oldDefault) {
    form.value.body = type === 'legacy' ? LEGACY_DEFAULT : COMPOSABLE_DEFAULT
  }
})

onMounted(load)
</script>
