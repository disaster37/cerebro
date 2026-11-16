<template>
  <div>
    <h4>Overview <small class="text-neutral-500">{{ clusterName }}</small></h4>
    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard
      v-else-if="error"
      class="mt-3 ring-1 ring-error-200"
      data-testid="overview-error"
    >
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">
        Connect to Cluster
      </UButton>
    </UCard>
    <div v-else-if="data">
      <div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UCard v-for="stat in stats" :key="stat.label" class="text-center">
          <p class="text-2xl font-semibold">{{ stat.value }}</p>
          <p class="text-sm text-neutral-500">{{ stat.label }}</p>
        </UCard>
      </div>
      <h5>Indices</h5>
      <UTable :data="(data.indices || [])" :columns="indicesColumns">
        <template #status-cell="{ row }">
          <UBadge v-if="row.original.closed" color="neutral" variant="soft">closed</UBadge>
          <UBadge v-else-if="row.original.special" color="info" variant="soft">system</UBadge>
          <UBadge v-else color="success" variant="soft">open</UBadge>
        </template>
      </UTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost, clusterName } = useCluster()
const { apiCall } = useAPI()
const data = ref<any>(null)
const loading = ref(true)
const error = ref('')

const stats = computed(() => {
  if (!data.value) return []
  return [
    { label: 'Nodes', value: data.value.number_of_nodes },
    { label: 'Active Shards', value: data.value.active_shards },
    { label: 'Unassigned', value: data.value.unassigned_shards },
    { label: 'Indices', value: data.value.total_indices },
  ]
})

const indicesColumns = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'doc_count', header: 'Docs' },
  { id: 'size', header: 'Size', cell: ({ row }: any) => formatBytes(row.original.total_size_in_bytes) },
  { id: 'status', header: 'Status' },
]

onMounted(async () => {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  const resp = await apiCall('POST', '/overview', { host: clusterHost.value })
  if (resp?.status === 200) {
    data.value = resp.body
  } else if (resp?.status === 401 || resp?.status === 403) {
    error.value = `Unauthorized (${resp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load cluster overview (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load cluster overview (${resp?.status || 'network error'}).`
  }
  loading.value = false
})

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>
