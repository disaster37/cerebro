<template>
  <div>
    <h4 class="mb-4 text-lg font-semibold">CAT API</h4>

    <div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_auto_auto]">
      <UFormField label="Preset endpoint">
        <USelect v-model="api" :items="presets" class="w-full" @update:model-value="onPreset" />
      </UFormField>
      <UFormField label="Custom path">
        <UInput v-model="api" placeholder="indices" class="w-full" />
      </UFormField>
      <UFormField label="Raw JSON">
        <USwitch v-model="rawJson" />
      </UFormField>
      <UFormField label="&nbsp;">
        <UButton :loading="loading" @click="execute">Execute</UButton>
      </UFormField>
    </div>

    <UAlert v-if="error" color="error" variant="soft" class="mb-3" :description="error" />

    <pre v-if="rawJson && result" class="overflow-x-auto rounded-lg border bg-neutral-100 p-4 text-xs dark:bg-neutral-900"><code>{{ JSON.stringify(result, null, 2) }}</code></pre>

    <CerebroTable
      v-else-if="rows.length"
      :data="rows"
      :columns="columns"
      filter-placeholder="Filter CAT rows..."
      :ui="{ th: 'px-1.5 py-1.5 text-[11px] font-semibold text-highlighted text-start align-middle whitespace-normal', td: 'px-1.5 py-1.5 text-[11px] text-muted align-middle whitespace-normal [overflow-wrap:anywhere]' }"
    />
    <p v-else-if="result" class="text-sm text-neutral-500">No rows returned.</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import CerebroTable from '../../components/CerebroTable.vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const presets = [
  'indices', 'nodes', 'shards', 'allocation', 'health', 'master', 'cluster_manager',
  'aliases', 'count', 'thread_pool', 'pending_tasks', 'recovery', 'segments',
  'templates', 'snapshots', 'repositories', 'nodeattrs', 'fielddata', 'plugins',
]

const api = ref('indices')
const result = ref<any>(null)
const rawJson = ref(false)
const loading = ref(false)
const error = ref('')

const rows = computed<any[]>(() => Array.isArray(result.value) ? result.value : [])

const columns = computed<CerebroColumnDef<any>[]>(() => {
  const first = rows.value[0]
  if (!first) return []
  return Object.keys(first).map((key) => ({
    id: key,
    header: key,
    accessorFn: (row: any) => smartValue(row[key]),
  }))
})

function smartValue(value: any): string | number {
  if (value === null || value === undefined) return ''
  if (typeof value === 'number') return value
  const text = String(value)
  const bytes = /^([\d.]+)\s*(b|kb|mb|gb|tb|pb)$/i.exec(text)
  if (bytes) {
    const units: Record<string, number> = { b: 1, kb: 1024, mb: 1024 ** 2, gb: 1024 ** 3, tb: 1024 ** 4, pb: 1024 ** 5 }
    return Number(bytes[1]) * (units[bytes[2]!.toLowerCase()] || 1)
  }
  const numeric = Number(text.replace(/,/g, ''))
  if (text.trim() !== '' && !Number.isNaN(numeric)) return numeric
  return text
}

function onPreset(value: string) {
  api.value = value
}

async function execute() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    return
  }
  loading.value = true
  error.value = ''
  const resp = await apiCall('POST', '/cat', { host: clusterHost.value, api: api.value }, { suppressErrorAlert: true })
  loading.value = false
  if (resp?.status === 200) {
    result.value = resp.body
  } else {
    error.value = extractEnvelopeError(resp?.body) || `CAT request failed (${resp?.status || 'network error'}).`
  }
}
</script>
