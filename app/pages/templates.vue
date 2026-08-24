<template>
  <div>
    <h4>Templates</h4>
    <table class="table table-sm">
      <thead><tr><th>Name</th><th>Patterns</th><th>Order</th><th>Actions</th></tr></thead>
      <tbody>
        <tr v-for="t in templates" :key="t.name">
          <td>{{ t.name }}</td>
          <td>{{ t.template?.index_patterns?.join(', ') }}</td>
          <td>{{ t.template?.order }}</td>
          <td><button class="btn btn-sm btn-danger" @click="del(t.name)">Delete</button></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const templates = ref<any[]>([])

onMounted(async () => {
  const resp = await apiCall('POST', '/templates', { host: clusterName.value })
  if (resp?.status === 200) templates.value = resp.body || []
})

async function del(name: string) {
  await apiCall('POST', '/templates/delete', { host: clusterName.value, name })
  const resp = await apiCall('POST', '/templates', { host: clusterName.value })
  if (resp?.status === 200) templates.value = resp.body || []
}
</script>
