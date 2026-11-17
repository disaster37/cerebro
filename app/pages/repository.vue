<template>
  <div>
    <h4>Repositories</h4>
    <UTable :data="repos" :columns="reposColumns" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const repos = ref<any[]>([])

const reposColumns = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'type', header: 'Type' },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/repositories', { host: clusterHost.value })
  if (resp?.status === 200) repos.value = resp.body || []
})
</script>
