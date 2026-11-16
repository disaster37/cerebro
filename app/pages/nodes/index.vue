<template>
  <div>
    <h4>Nodes</h4>
    <UTable :data="nodes" :columns="nodesColumns">
      <template #role-cell="{ row }">
        <UBadge v-if="row.original.master" color="primary" variant="solid">M</UBadge>
        <UBadge v-if="row.original.data" color="success" variant="solid">D</UBadge>
        <UBadge v-if="row.original.ingest" color="info" variant="solid">I</UBadge>
        <UBadge v-if="row.original.coordinating" color="neutral" variant="solid">C</UBadge>
      </template>
    </UTable>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const nodes = ref<any[]>([])

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
  const resp = await apiCall('POST', '/nodes', { host: clusterName.value })
  if (resp?.status === 200 && Array.isArray(resp.body)) nodes.value = resp.body
})

function formatBytes(bytes: number): string {
  if (!bytes) return '0'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>
