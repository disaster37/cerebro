<template>
  <div>
    <h4>Repositories</h4>
    <UTable :data="repos" :columns="reposColumns" />
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const repos = ref<any[]>([])

const reposColumns = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'type', header: 'Type' },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/repositories', { host: clusterName.value })
  if (resp?.status === 200) repos.value = resp.body || []
})
</script>
