<template>
  <div>
    <h4>Nodes</h4>
    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard
      v-else-if="error"
      class="mt-3 ring-1 ring-error-200"
      data-testid="nodes-error"
    >
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">
        Connect to Cluster
      </UButton>
    </UCard>
    <UTable v-else-if="nodes.length" :data="nodes" :columns="nodesColumns">
      <template #role-cell="{ row }">
        <UBadge v-if="row.original.master" color="primary" variant="solid">M</UBadge>
        <UBadge v-if="row.original.data" color="success" variant="solid">D</UBadge>
        <UBadge v-if="row.original.ingest" color="info" variant="solid">I</UBadge>
        <UBadge v-if="row.original.coordinating" color="neutral" variant="solid">C</UBadge>
      </template>
    </UTable>
    <p v-else class="mt-3 text-sm text-neutral-500">No nodes found.</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const nodes = ref<any[]>([])
const loading = ref(true)
const error = ref('')

const nodesColumns = [
  { id: 'name', header: 'Name', cell: ({ row }: any) => `${row.original.name}${row.original.current_master ? ' *' : ''}` },
  { id: 'role', header: 'Role' },
  { accessorKey: 'cpu_percent', header: 'CPU', cell: ({ row }: any) => `${row.original.cpu_percent}%` },
  { id: 'heap', header: 'Heap', cell: ({ row }: any) => `${formatBytes(row.original.heap?.used)} / ${formatBytes(row.original.heap?.max)}` },
  { accessorKey: 'disk', header: 'Disk', cell: ({ row }: any) => `${row.original.disk?.used_percent}%` },
  { id: 'load', header: 'Load', cell: ({ row }: any) => row.original.load_average?.toFixed(2) },
  { accessorKey: 'es_version', header: 'Version' },
]

onMounted(async () => {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  const resp = await apiCall('POST', '/nodes', { host: clusterHost.value })
  if (resp?.status === 200 && Array.isArray(resp.body)) {
    nodes.value = resp.body
  } else if (resp?.status === 401 || resp?.status === 403) {
    error.value = `Unauthorized (${resp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load cluster nodes (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load cluster nodes (${resp?.status || 'network error'}).`
  }
  loading.value = false
})

function formatBytes(bytes: number): string {
  if (!bytes) return '0'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>
