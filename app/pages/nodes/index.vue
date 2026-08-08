<template>
  <div>
    <h4>Nodes</h4>
    <table class="table table-sm table-striped" v-if="nodes.length">
      <thead><tr><th>Name</th><th>Role</th><th>CPU</th><th>Heap</th><th>Disk</th><th>Load</th><th>Version</th></tr></thead>
      <tbody>
        <tr v-for="n in nodes" :key="n.id" :class="{ 'table-primary': n.current_master }">
          <td>{{ n.name }}{{ n.current_master ? ' *' : '' }}</td>
          <td>
            <span v-if="n.master" class="badge bg-primary">M</span>
            <span v-if="n.data" class="badge bg-success">D</span>
            <span v-if="n.ingest" class="badge bg-info">I</span>
            <span v-if="n.coordinating" class="badge bg-secondary">C</span>
          </td>
          <td>{{ n.cpu_percent }}%</td>
          <td>{{ formatBytes(n.heap?.used) }} / {{ formatBytes(n.heap?.max) }}</td>
          <td>{{ n.disk?.used_percent }}%</td>
          <td>{{ n.load_average?.toFixed(2) }}</td>
          <td>{{ n.es_version }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const nodes = ref<any[]>([])

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
