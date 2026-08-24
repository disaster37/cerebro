<template>
  <div>
    <h4>Overview <small class="text-muted">{{ clusterName }}</small></h4>
    <div v-if="loading" class="text-center p-5">
      <div class="spinner-border" role="status"></div>
    </div>
    <div v-else-if="data">
      <div class="row mb-3">
        <div class="col-md-3" v-for="stat in stats" :key="stat.label">
          <div class="card">
            <div class="card-body text-center">
              <h5 class="card-title">{{ stat.value }}</h5>
              <small class="text-muted">{{ stat.label }}</small>
            </div>
          </div>
        </div>
      </div>
      <h5>Indices</h5>
      <table class="table table-sm table-striped">
        <thead><tr><th>Name</th><th>Docs</th><th>Size</th><th>Status</th></tr></thead>
        <tbody>
          <tr v-for="idx in (data.indices || [])" :key="idx.name" :class="{ 'table-warning': idx.unhealthy }">
            <td>{{ idx.name }}</td>
            <td>{{ idx.doc_count }}</td>
            <td>{{ formatBytes(idx.total_size_in_bytes) }}</td>
            <td>
              <span v-if="idx.closed" class="badge bg-secondary">closed</span>
              <span v-else-if="idx.special" class="badge bg-info">system</span>
              <span v-else class="badge bg-success">open</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const data = ref<any>(null)
const loading = ref(true)
const stats = computed(() => {
  if (!data.value) return []
  return [
    { label: 'Nodes', value: data.value.number_of_nodes },
    { label: 'Active Shards', value: data.value.active_shards },
    { label: 'Unassigned', value: data.value.unassigned_shards },
    { label: 'Indices', value: data.value.total_indices },
  ]
})

onMounted(async () => {
  const resp = await apiCall('POST', '/overview', { host: clusterName.value })
  if (resp?.status === 200) data.value = resp.body
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
