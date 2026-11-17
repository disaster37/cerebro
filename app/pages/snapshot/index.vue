<template>
  <div>
    <h4>Snapshots</h4>
    <UTable :data="snapshots" :columns="snapshotsColumns" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const snapshots = ref<any[]>([])
const repos = ref<string[]>([])

const snapshotsColumns = [
  { accessorKey: 'snapshot', header: 'Snapshot' },
  { accessorKey: 'state', header: 'State' },
  { id: 'indices', header: 'Indices', cell: ({ row }: any) => row.original.indices?.join(', ') },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/snapshots', { host: clusterHost.value })
  if (resp?.status === 200) {
    repos.value = resp.body?.repositories || []
    if (repos.value[0]) await loadSnaps(repos.value[0])
  }
})

async function loadSnaps(repo: string) {
  const resp = await apiCall('POST', '/snapshots/load', { host: clusterHost.value, repository: repo })
  if (resp?.status === 200) snapshots.value = resp.body || []
}
</script>
