<template>
  <div>
    <h4 class="mb-4 text-lg font-semibold">Analysis</h4>

    <div class="mb-4 flex gap-2">
      <UButton size="sm" :color="mode === 'field' ? 'primary' : 'neutral'" :variant="mode === 'field' ? 'solid' : 'outline'" @click="mode = 'field'">
        By Field
      </UButton>
      <UButton size="sm" :color="mode === 'analyzer' ? 'primary' : 'neutral'" :variant="mode === 'analyzer' ? 'solid' : 'outline'" @click="mode = 'analyzer'">
        By Analyzer
      </UButton>
    </div>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
      <UFormField label="Index">
        <USelect v-model="selectedIndex" :items="indices" placeholder="Select..." class="w-full" @update:model-value="onIndexChange" />
      </UFormField>
      <UFormField v-if="mode === 'field'" label="Field">
        <USelect v-model="selectedField" :items="fields" placeholder="Select..." class="w-full" />
      </UFormField>
      <UFormField v-else label="Analyzer">
        <USelect v-model="selectedAnalyzer" :items="analyzers" placeholder="Select..." class="w-full" />
      </UFormField>
      <UFormField label="Text">
        <UInput v-model="text" class="w-full" />
      </UFormField>
      <UFormField label="&nbsp;">
        <UButton block :loading="analyzing" @click="analyze">Analyze</UButton>
      </UFormField>
    </div>

    <UAlert v-if="error" color="error" variant="soft" class="mt-3" :description="error" />

    <div v-if="tokens.length" class="mt-4">
      <CerebroTable
        :data="filteredTokens"
        :columns="columns"
        :default-sort="[{ id: 'position', desc: false }]"
        filter-placeholder="Filter tokens..."
      >
        <template #filters>
          <USelect v-model="tokenTypeFilter" :items="tokenTypeOptions" size="sm" class="w-36" />
        </template>
      </CerebroTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CerebroTable from '../../components/CerebroTable.vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'

interface Token {
  token: string
  position: number
  type: string
  start_offset: number
  end_offset: number
}

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const indices = ref<string[]>([])
const fields = ref<string[]>([])
const analyzers = ref<string[]>([])
const selectedIndex = ref('')
const selectedField = ref('')
const selectedAnalyzer = ref('')
const text = ref('')
const tokens = ref<Token[]>([])
const mode = ref<'field' | 'analyzer'>('field')
const tokenTypeFilter = ref('all')
const analyzing = ref(false)
const error = ref('')

const tokenTypeOptions = computed(() => [
  { label: 'All types', value: 'all' },
  ...[...new Set(tokens.value.map((t) => t.type))].map((type) => ({ label: type, value: type })),
])

const filteredTokens = computed(() => tokenTypeFilter.value === 'all'
  ? tokens.value
  : tokens.value.filter((t) => t.type === tokenTypeFilter.value))

const columns: CerebroColumnDef<Token>[] = [
  { id: 'position', header: 'Position', accessorFn: (t) => t.position ?? 0, class: 'whitespace-nowrap' },
  { id: 'token', header: 'Token', accessorKey: 'token' },
  { id: 'type', header: 'Type', accessorKey: 'type', class: 'whitespace-nowrap' },
  { id: 'start_offset', header: 'Start Offset', accessorFn: (t) => t.start_offset ?? 0, class: 'whitespace-nowrap' },
  { id: 'end_offset', header: 'End Offset', accessorFn: (t) => t.end_offset ?? 0, class: 'whitespace-nowrap' },
]

async function loadIndices() {
  if (!clusterHost.value) return
  const resp = await apiCall('POST', '/analysis/indices', { host: clusterHost.value })
  if (resp?.status === 200) indices.value = resp.body || []
}

async function onIndexChange() {
  if (!selectedIndex.value) return
  const [fieldsResp, analyzersResp] = await Promise.all([
    apiCall('POST', '/analysis/fields', { host: clusterHost.value, index: selectedIndex.value }),
    apiCall('POST', '/analysis/analyzers', { host: clusterHost.value, index: selectedIndex.value }),
  ])
  if (fieldsResp?.status === 200) fields.value = fieldsResp.body || []
  if (analyzersResp?.status === 200) analyzers.value = analyzersResp.body || []
}

async function analyze() {
  analyzing.value = true
  error.value = ''
  const url = mode.value === 'field' ? '/analysis/analyze/field' : '/analysis/analyze/analyzer'
  const body: Record<string, any> = {
    host: clusterHost.value,
    index: selectedIndex.value,
    text: text.value,
  }
  if (mode.value === 'field') body.field = selectedField.value
  else body.analyzer = selectedAnalyzer.value

  const resp = await apiCall('POST', url, body, { suppressErrorAlert: true })
  analyzing.value = false
  if (resp?.status === 200) {
    tokens.value = resp.body || []
  } else {
    error.value = extractEnvelopeError(resp?.body) || `Analysis failed (${resp?.status || 'network error'}).`
  }
}

onMounted(loadIndices)
</script>
